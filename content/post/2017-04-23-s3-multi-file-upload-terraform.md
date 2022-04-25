---
date: "2017-04-23T00:00:00Z"
tags: ["aws", "terraform", "s3"]
title: S3 Multi-File upload with Terraform
---

Hosting a static website with S3 is really easy, especially from terraform:

First off, we want a public readable S3 bucket policy, but we want to apply this only to one specific bucket.  To achive that we can use Terraform's `template_file` data block to merge in a value:


```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "PublicReadGetObject",
      "Effect": "Allow",
      "Principal": "*",
      "Action": [
        "s3:GetObject"
      ],
      "Resource": [
        "arn:aws:s3:::${bucket_name}/*"
      ]
    }
  ]
}
```

As you can see the interpolation syntax is pretty much the same as how you use variables in terraform itself.  Next we define a `template_file` to do the transformation.  As the bucket name is going to be used many times, we extract that into a `variable` block also:

```cmake
variable "bucket" {
  default = "examplebucket"
}

data "template_file" "s3_public_policy" {
  template = "${file("policies/s3-public.json")}"
  vars {
    bucket_name = "${var.bucket}"
  }
}
```

Next we want to create the S3 bucket and set it to be a static website, which we can do using the `website` sub block.  For added usefulness, we will also define an `output` to show the website url on the command line:

```cmake
resource "aws_s3_bucket" "static_site" {
  bucket = "${var.bucket}"
  acl = "public-read"
  policy = "${data.template_file.s3_public_policy.rendered}"

  website {
    index_document = "index.html"
  }
}

output "url" {
  value = "${aws_s3_bucket.static_site.bucket}.s3-website-${var.region}.amazonaws.com"
}
```

## Single File Upload

If you just want one file in the website (say the `index.html` file), then you can add the following block.  Just make sure the `key` property matches the `index_document` name in the `aws_s3_bucket` block.

```cmake
resource "aws_s3_bucket_object" "index" {
  bucket = "${aws_s3_bucket.static_site.bucket}"
  key = "index.html"
  source = "src/index.html"
  content_type = "text/html"
  etag = "${md5(file("src/index.html"))}"
}
```

## Multi File Upload

Most websites need more than one file to be useful, and while we could write out an `aws_s3_bucket_object` block for every file, that seems like a lot of effort.  Other options include manually uploading the files to S3, or using the aws cli to do it.  While both methods work, they're error prone - you need to specify the `content_type` for each file for them to load properly, and you can't change this property once a file is uploaded.

To get around this, I add one more variable to my main terraform file, and generate a second file with all the `aws_s3_bucket_object` blocks in I need.

The added `variable` is a lookup for mime types:

```cmake
variable "mime_types" {
  default = {
    htm = "text/html"
    html = "text/html"
    css = "text/css"
    js = "application/javascript"
    map = "application/javascript"
    json = "application/json"
  }
}
```

I then create a shell script which will write a new file containing a `aws_s3_bucket_object` block for each file in the `src` directory:

```bash
#! /bin/sh

SRC="src/"
TF_FILE="files.tf"
COUNT=0

cat > $TF_FILE ''

find $SRC -iname '*.*' | while read path; do

    cat >> $TF_FILE << EOM

resource "aws_s3_bucket_object" "file_$COUNT" {
  bucket = "\${aws_s3_bucket.static_site.bucket}"
  key = "${path#$SRC}"
  source = "$path"
  content_type = "\${lookup(var.mime_types, "${path##*.}")}"
  etag = "\${md5(file("$path"))}"
}
EOM

    COUNT=$(expr $COUNT + 1)

done
```

Now when I want to publish a static site, I just have to make sure I run `./files.sh` once before my `terraform plan` and `terraform apply` calls.

## Caveats

This technique has one major drawback: it doesn't work well with updating an existing S3 bucket.  It won't remove files which are no longer in the terraform files, and can't detect file moves.

However, if you're happy with a call to `terraform destroy` before applying, this will work fine.  I use it for a number of test sites which I don't tend to leave online very long, and for scripted aws infrastructure that I give out to other people so they can run their own copy.
