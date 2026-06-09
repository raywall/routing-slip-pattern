provider "aws" {
  region = var.aws_region
}

locals {
  layer_source_dir = coalesce(var.layer_source_dir, "${path.module}/layer")
  layer_zip        = "${path.module}/${var.layer_name}.zip"
  example_layers   = concat([aws_lambda_layer_version.routing_slip.arn], var.extra_layer_arns)
  example_environment = merge({
    ROUTING_SLIP_CONFIG_PATH   = "/opt/routing-slip/config/config.yaml"
    ROUTING_SLIP_WORKFLOW_PATH = "/opt/routing-slip/workflows/workflow.yaml"
  }, var.example_lambda_environment)
}

data "archive_file" "routing_slip_layer" {
  type        = "zip"
  source_dir  = local.layer_source_dir
  output_path = local.layer_zip
}

resource "aws_lambda_layer_version" "routing_slip" {
  filename                 = data.archive_file.routing_slip_layer.output_path
  layer_name               = var.layer_name
  description              = var.layer_description
  source_code_hash         = data.archive_file.routing_slip_layer.output_base64sha256
  compatible_runtimes      = var.compatible_runtimes
  compatible_architectures = var.compatible_architectures
}

data "aws_iam_policy_document" "example_assume_role" {
  count = var.create_example_lambda && var.example_lambda_role_arn == "" ? 1 : 0

  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "example" {
  count = var.create_example_lambda && var.example_lambda_role_arn == "" ? 1 : 0

  name               = "${var.example_lambda_name}-role"
  assume_role_policy = data.aws_iam_policy_document.example_assume_role[0].json
}

resource "aws_iam_role_policy_attachment" "example_basic" {
  count = var.create_example_lambda && var.example_lambda_role_arn == "" ? 1 : 0

  role       = aws_iam_role.example[0].name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

resource "aws_lambda_function" "example" {
  count = var.create_example_lambda ? 1 : 0

  function_name    = var.example_lambda_name
  role             = var.example_lambda_role_arn != "" ? var.example_lambda_role_arn : aws_iam_role.example[0].arn
  filename         = var.example_lambda_package_file
  source_code_hash = filebase64sha256(var.example_lambda_package_file)
  handler          = "bootstrap"
  runtime          = "provided.al2023"
  architectures    = [var.compatible_architectures[0]]
  timeout          = var.example_lambda_timeout
  memory_size      = var.example_lambda_memory_size
  layers           = local.example_layers

  environment {
    variables = local.example_environment
  }

  depends_on = [aws_iam_role_policy_attachment.example_basic]
}
