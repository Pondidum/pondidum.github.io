---
date: "2017-07-12T00:00:00Z"
tags: ["aws", "terraform", "s3"]
title: Terraform, Kinesis Streams, Lambda and IAM problems
---

I hit an problem the recently with Terraform, when I was trying to hook up a Lambda Trigger to a Kinesis stream.  Both the lambda itself, and the stream creation succeeded within Terraform, but the trigger would just stay stuck on "creating..." for at least 5 minutes, before I got bored of waiting and killed the process.  Several attempts at doing this had the same issue.

The code looked something along the lines of this:

```bash
data "archive_file" "consumer_lambda" {
  type = "zip"
  source_dir = "./js/consumer"
  output_path = "./build/consumer.zip"
}

resource "aws_lambda_function" "kinesis_consumer" {
  filename = "${data.archive_file.consumer_lambda.output_path}"
  function_name = "kinesis_consumer"
  role = "${aws_iam_role.consumer_role.arn}"
  handler = "index.handler"
  runtime = "nodejs6.10"
  source_code_hash = "${base64sha256(file("${data.archive_file.consumer_lambda.output_path}"))}"
  timeout = 300 # 5 mins
}

resource "aws_kinesis_stream" "replay_stream" {
  name = "replay_stream"
  shard_count = 1
}

resource "aws_lambda_event_source_mapping" "kinesis_replay_lambda" {
  event_source_arn = "${aws_kinesis_stream.replay_stream.arn}"
  function_name = "${aws_lambda_function.kinesis_consumer.arn}"
  starting_position = "TRIM_HORIZON"
}

resource "aws_iam_role" "consumer_role" {
  name = "consumer_role"
  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": ["lambda.amazonaws.com"]
      },
      "Effect": "Allow",
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "consumer_role_policy" {
  name = "consumer_role_policy"
  role = "${aws_iam_role.consumer_role.id}"
  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "Stmt1493060054000",
      "Effect": "Allow",
      "Action": ["lambda:InvokeAsync", "lambda:InvokeFunction"],
      "Resource": ["arn:aws:lambda:*:*:*"]
    },
    {
      "Effect": "Allow",
      "Action": ["s3:GetObject*", "s3:PutObject*"],
      "Resource": ["arn:aws:s3:::*"]
    }
}
EOF
}
```

I decided to try creating the trigger manually in AWS, which gave me the following error:

> There was an error creating the trigger: Cannot access stream arn:aws:kinesis:eu-west-1:586732038447:stream/test. Please ensure the role can perform the **GetRecords**, **GetShardIterator**, **DescribeStream**, and **ListStreams** Actions on your stream in IAM.

All I had to do to fix this was to change my `consumer_role_policy` to include the relevant permissions:

```json
{
    "Effect": "Allow",
    "Action": [
        "kinesis:DescribeStream",
        "kinesis:GetShardIterator",
        "kinesis:GetRecords",
        "kinesis:ListStreams",
        "kinesis:PutRecords"
    ],
    "Resource": "arn:aws:kinesis:*:*:*"
}
```

## Takeaways

* **Terraform could do with better errors** - preferably in nice red text telling me I am doing things wrong!
* **AWS told me exactly what was needed** - Good error messages in AWS, so no need to spend hours googling which permissions would be needed.
