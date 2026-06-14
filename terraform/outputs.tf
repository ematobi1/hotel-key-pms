output "cluster_endpoint" {
  description = "EKS cluster API endpoint"
  value       = module.eks.cluster_endpoint
  sensitive   = false
}

output "cluster_name" {
  description = "EKS cluster name"
  value       = module.eks.cluster_name
}

output "cluster_certificate_authority" {
  description = "EKS cluster certificate authority data (base64)"
  value       = module.eks.cluster_certificate_authority
  sensitive   = true
}

output "database_address" {
  description = "RDS PostgreSQL endpoint address"
  value       = module.rds.db_address
}

output "database_port" {
  description = "RDS PostgreSQL port"
  value       = module.rds.db_port
}

output "database_name" {
  description = "RDS database name"
  value       = module.rds.db_name
}

output "redis_primary_endpoint" {
  description = "ElastiCache Redis primary endpoint"
  value       = module.elasticache.primary_endpoint
}

output "vpc_id" {
  description = "VPC ID"
  value       = module.vpc.vpc_id
}

output "private_subnet_ids" {
  description = "Private subnet IDs"
  value       = module.vpc.private_subnet_ids
}
