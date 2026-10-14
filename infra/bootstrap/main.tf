terraform {
  required_version = ">= 1.12, < 2.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }
}
provider "aws" {
  region = "ap-south-1"
  default_tags {
    tags = { Project = "developer-portal", ManagedBy = "Terraform" }
  }
}
data "aws_caller_identity" "current" {}
locals {
  operator_name = "portal-operator"
  operator_arn  = "arn:aws:iam::316865756899:user/${local.operator_name}"
}

# Console passwords are set by the owner in AWS, outside Terraform state.
resource "aws_iam_user" "operator" {
  name = local.operator_name
  lifecycle {
    precondition {
      condition     = data.aws_caller_identity.current.account_id == "316865756899"
      error_message = "The AWS profile is signed into a different account."
    }
  }
}
resource "aws_iam_user_policy_attachment" "sign_in" {
  user       = aws_iam_user.operator.name
  policy_arn = "arn:aws:iam::aws:policy/SignInLocalDevelopmentAccess"
}
resource "aws_iam_user_policy" "operator" {
  name = "developer-portal-access"
  user = aws_iam_user.operator.name
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid      = "DescribePracticeCluster"
        Effect   = "Allow"
        Action   = ["eks:DescribeCluster"]
        Resource = "arn:aws:eks:ap-south-1:316865756899:cluster/developer-portal"
      },
      {
        Sid      = "ChangeOwnConsolePassword"
        Effect   = "Allow"
        Action   = ["iam:ChangePassword"]
        Resource = local.operator_arn
      },
      {
        Sid      = "ReadPasswordPolicy"
        Effect   = "Allow"
        Action   = ["iam:GetAccountPasswordPolicy"]
        Resource = "*"
      }
    ]
  })
}
output "operator_arn" { value = aws_iam_user.operator.arn }
output "console_sign_in_url" { value = "https://316865756899.signin.aws.amazon.com/console" }
