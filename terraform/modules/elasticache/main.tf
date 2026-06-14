resource "aws_elasticache_subnet_group" "main" {
  name       = "hotel-key-${var.environment}-redis"
  subnet_ids = var.private_subnet_ids
}

resource "aws_security_group" "redis" {
  name        = "hotel-key-${var.environment}-redis-sg"
  description = "ElastiCache Redis security group"
  vpc_id      = var.vpc_id

  ingress {
    from_port       = 6379
    to_port         = 6379
    protocol        = "tcp"
    security_groups = [var.eks_security_group]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = { Name = "hotel-key-${var.environment}-redis-sg" }
}

resource "aws_kms_key" "redis" {
  description             = "ElastiCache encryption key for hotel-key-${var.environment}"
  deletion_window_in_days = 7
  enable_key_rotation     = true
}

resource "aws_elasticache_replication_group" "main" {
  replication_group_id       = "hotel-key-${var.environment}"
  description                = "Hotel Key PMS Redis cluster"
  node_type                  = var.node_type
  num_cache_clusters         = var.num_cache_nodes
  port                       = 6379
  subnet_group_name          = aws_elasticache_subnet_group.main.name
  security_group_ids         = [aws_security_group.redis.id]
  parameter_group_name       = aws_elasticache_parameter_group.main.name
  engine_version             = "7.1"
  at_rest_encryption_enabled = true
  transit_encryption_enabled = true
  kms_key_id                 = aws_kms_key.redis.arn
  automatic_failover_enabled = var.num_cache_nodes > 1
  multi_az_enabled           = var.num_cache_nodes > 1
  maintenance_window         = "sun:05:00-sun:06:00"
  snapshot_retention_limit   = 3
  snapshot_window            = "04:00-05:00"

  log_delivery_configuration {
    destination      = aws_cloudwatch_log_group.redis_slow.name
    destination_type = "cloudwatch-logs"
    log_format       = "json"
    log_type         = "slow-log"
  }

  tags = { Name = "hotel-key-${var.environment}-redis" }
}

resource "aws_elasticache_parameter_group" "main" {
  name   = "hotel-key-${var.environment}-redis71"
  family = "redis7"

  parameter {
    name  = "maxmemory-policy"
    value = "allkeys-lru"
  }
  parameter {
    name  = "slowlog-log-slower-than"
    value = "10000"
  }
}

resource "aws_cloudwatch_log_group" "redis_slow" {
  name              = "/aws/elasticache/hotel-key-${var.environment}/slow-log"
  retention_in_days = 14
}
