output "cluster_name" { value = aws_eks_cluster.portal.name }
output "aws_region" { value = var.aws_region }
output "table_name" { value = aws_dynamodb_table.projects.name }
output "project_role_arn" { value = aws_iam_role.project_service.arn }
output "repositories" { value = { for name, repo in aws_ecr_repository.services : name => repo.repository_url } }
