variable "region" {
  description = "The AWS region."
  default     = "eu-central-1"
}

variable "name" {
  description = "Application Name"
  type        = string
}

variable "environment" {
  description = "Env"
  default     = "dev"
}

locals {
  description = "Aplication Name"
  app_name    = "${var.name}-${var.environment}"
}


variable "shared_credentials_file" {
  description = "Shared credentials file"
  type        = string
  sensitive   = true
}
