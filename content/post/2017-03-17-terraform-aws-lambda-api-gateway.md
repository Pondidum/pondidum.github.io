---
layout: post
title: Using Terraform to setup AWS API-Gateway and Lambda
tags: c# nodejs aws terraform lambda apigateway rest
---

I have been writing simple webhook type applications using [Claudiajs](https://claudiajs.com/), which in behind the scenes is using Aws's Lambda and Api Gateway to make things happen, but I really wanted to understand what exactly it was doing for me, and how I could achieve the same results using [Terraform](https://terraform.io).

### The Lambda Function

I started off with a simple NodeJS function, in a file called `index.js`

```javascript
exports.handler = function(event, context, callback) {
  callback(null, {
    statusCode: '200',
    body: JSON.stringify({ 'message': 'hello world' }),
    headers: {
      'Content-Type': 'application/json',
    },
  });
};
```

First thing to note about this function is the 2nd argument passed to `callback`: **this maps to the whole response object not just the body**.  If you try and just run `callback(null, { message: 'hello world' })`, when called from the API Gateway, you will get the following error in your CloudWatch logs, and not a lot of help on Google:

> Execution failed due to configuration error: "Malformed Lambda proxy response"

## Terraform

We want to upload a zip file containing all our lambda's code, which in this case is just the `index.js` file.  While this could be done by generating the zip file with a gulp script or manually, we can just get terraform to do this for us, by using the [archive_file data source](https://www.terraform.io/docs/providers/archive/d/archive_file.html):

```cmake
data "archive_file" "lambda" {
  type = "zip"
  source_file = "index.js"
  output_path = "lambda.zip"
}

resource "aws_lambda_function" "example_test_function" {
  filename = "${data.archive_file.lambda.output_path}"
  function_name = "example_test_function"
  role = "${aws_iam_role.example_api_role.arn}"
  handler = "index.handler"
  runtime = "nodejs4.3"
  source_code_hash = "${base64sha256(file("${data.archive_file.lambda.output_path}"))}"
  publish = true
}
```
By using the `source_code_hash` property, Terraform can detect when the zip file has changed, and thus know whether to re-upload the function when you call `terraform apply`.

We also need an IAM role for the function to run under.  While the policy could be written inline, but I have found it more expressive to have a separate file for the role policy:

```cmake
resource "aws_iam_role" "example_api_role" {
  name = "example_api_role"
  assume_role_policy = "${file("policies/lambda-role.json")}"
}
```

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": [
          "lambda.amazonaws.com",
          "apigateway.amazonaws.com"
        ]
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
```

That's the lambda done - you can login to the AWS Console, setup a test event and execute it if you want :)


### Creating the Api Gateway

We are going to create a simple api, with one endpoint (or resource, in AWS terminology).

First we need to define an api root:

```cmake
resource "aws_api_gateway_rest_api" "example_api" {
  name = "ExampleAPI"
  description = "Example Rest Api"
}
```

And then a resource to represent the `/messages` endpoint, and a method to handle `POST`:

```cmake
resource "aws_api_gateway_resource" "example_api_resource" {
  rest_api_id = "${aws_api_gateway_rest_api.example_api.id}"
  parent_id = "${aws_api_gateway_rest_api.example_api.root_resource_id}"
  path_part = "messages"
}

resource "aws_api_gateway_method" "example_api_method" {
  rest_api_id = "${aws_api_gateway_rest_api.example_api.id}"
  resource_id = "${aws_api_gateway_resource.example_api_resource.id}"
  http_method = "POST"
  authorization = "NONE"
}
```

The `aws_api_gateway_resource` can be attached to other `aws_api_gateway_resource`s rather than to the api root too, allowing for multi level routes.  You can do this by changing the `parent_id` property to point to another `aws_api_gateway_resource.id`.

Now we need add an integration between the api and lambda:

```cmake
resource "aws_api_gateway_integration" "example_api_method-integration" {
  rest_api_id = "${aws_api_gateway_rest_api.example_api.id}"
  resource_id = "${aws_api_gateway_resource.example_api_resource.id}"
  http_method = "${aws_api_gateway_method.example_api_method.http_method}"
  type = "AWS_PROXY"
  uri = "arn:aws:apigateway:${var.region}:lambda:path/2015-03-31/functions/arn:aws:lambda:${var.region}:${var.account_id}:function:${aws_lambda_function.example_test_function.function_name}/invocations"
  integration_http_method = "POST"
}
```

Finally a couple of deployment stages, and an output variable for each to let you know the api's urls:

```cmake
resource "aws_api_gateway_deployment" "example_deployment_dev" {
  depends_on = [
    "aws_api_gateway_method.example_api_method",
    "aws_api_gateway_integration.example_api_method-integration"
  ]
  rest_api_id = "${aws_api_gateway_rest_api.example_api.id}"
  stage_name = "dev"
}

resource "aws_api_gateway_deployment" "example_deployment_prod" {
  depends_on = [
    "aws_api_gateway_method.example_api_method",
    "aws_api_gateway_integration.example_api_method-integration"
  ]
  rest_api_id = "${aws_api_gateway_rest_api.example_api.id}"
  stage_name = "api"
}

output "dev_url" {
  value = "https://${aws_api_gateway_deployment.example_deployment_dev.rest_api_id}.execute-api.${var.region}.amazonaws.com/${aws_api_gateway_deployment.example_deployment_dev.stage_name}"
}

output "prod_url" {
  value = "https://${aws_api_gateway_deployment.example_deployment_prod.rest_api_id}.execute-api.${var.region}.amazonaws.com/${aws_api_gateway_deployment.example_deployment_prod.stage_name}"
}
```

The two output variables will cause terraform to output the paths when you call `terraform apply`, or afterwards when you call `terraform output dev_url`.  Great for scripts which need to know the urls!

### Run it!

You can now call your url and see a friendly hello world message:

```bash
curl -X POST -H "Content-Type: application/json" "YOUR_DEV_OR_PROD_URL"
```

## Switching to C\#

Switching to a C#/dotnetcore lambda is very straight forward from here.  We just need to change the `aws_lambda_function`'s runtime and handler properties, and change the `archive_file` to use `source_dir` rather than `source_file`:

```cmake
data "archive_file" "lambda" {
  type = "zip"
  source_dir = "./src/published"
  output_path = "lambda.zip"
}

resource "aws_lambda_function" "example_test_function" {
  filename = "${data.archive_file.lambda.output_path}"
  function_name = "example_test_function"
  role = "${aws_iam_role.example_api_role.arn}"
  handler = "ExampleLambdaApi::ExampleLambdaApi.Handler::Handle"
  runtime = "dotnetcore1.0"
  source_code_hash = "${base64sha256(file("${data.archive_file.lambda.output_path}"))}"
  publish = true
}
```

Note the `handler` property is in the form `AssemblyName::FullyQualifiedTypeName::MethodName`.

For our C# project, we need the following two nugets:

```bash
Amazon.Lambda.APIGatewayEvents
Amazon.Lambda.Serialization.Json
```
And the only file in our project looks like so:

```csharp
namespace ExampleLambdaApi
{
  public class Handler
  {
    [LambdaSerializer(typeof(JsonSerializer))]
    public APIGatewayProxyResponse Handle(APIGatewayProxyRequest apigProxyEvent)
    {
      return new APIGatewayProxyResponse
      {
        Body = apigProxyEvent.Body,
        StatusCode = 200,
      };
    }
  }
}
```

One thing worth noting is that the first time a C# function is called it takes a long time - in the region of 5-6 seconds.  Subsequent invocations are in the 200ms region.

All the code for this demo can be found on my [GitHub](https://github.com/pondidum/), in the [terraform-demos repository](https://github.com/Pondidum/Terraform-Demos/tree/master/api-lambda).
