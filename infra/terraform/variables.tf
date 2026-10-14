variable "aws_region" {
  type        = string
  description = "AWS region selected for this deployment."
}
variable "expected_account_id" {
  type    = string
  default = "316865756899"
}
variable "project_name" {
  type    = string
  default = "developer-portal"
  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{2,24}$", var.project_name))
    error_message = "Use 3-25 lowercase letters, numbers, or hyphens, starting with a letter."
  }
}
variable "kubernetes_version" {
  type    = string
  default = "1.36"
}
variable "admin_principal_arn" {
  type        = string
  description = "IAM role/user ARN for kubectl access; do not use an STS assumed-role session ARN."
  validation {
    condition     = can(regex("^arn:aws:iam::[0-9]{12}:(role|user)/.+$", var.admin_principal_arn))
    error_message = "Supply an IAM role or user ARN."
  }
}
variable "admin_cidr" {
  type        = string
  description = "Administrator public IPv4 address with /32, for example 203.0.113.10/32."
  validation {
    condition     = can(cidrhost(var.admin_cidr, 0)) && can(regex("/32$", var.admin_cidr))
    error_message = "Restrict the Kubernetes API to your public IPv4 address using /32."
  }
}
variable "node_instance_type" {
  type    = string
  default = "t3.medium"
}
