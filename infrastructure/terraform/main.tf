
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 3.22"
    }
  }
}


provider "aws" {
  profile                 = "default"
  region                  = var.region
  shared_credentials_file = var.shared_credentials_file
}

resource "aws_sqs_queue" "inbound-queue" {
  name                      = "inbound-queue-${var.environment}"
  delay_seconds             = 0
  max_message_size          = 262144
  message_retention_seconds = 86400
  receive_wait_time_seconds = 10

  tags = {
    app = local.app_name
  }
}
resource "aws_sqs_queue" "outbound-queue" {
  name                      = "outbound-queue-${var.environment}"
  delay_seconds             = 0
  max_message_size          = 262144
  message_retention_seconds = 86400
  receive_wait_time_seconds = 10

  tags = {
    app = local.app_name
  }
}
resource "aws_sqs_queue" "results-queue" {
  name                      = "results-queue-${var.environment}"
  delay_seconds             = 0
  max_message_size          = 262144
  message_retention_seconds = 86400
  receive_wait_time_seconds = 10

  tags = {
    app = local.app_name
  }
}

# resource "aws_iam_role" "sqs_access" {
#   name               = "sqs-role"
#   assume_role_policy = "${file("assumerolepolicy.json")}"
# }
