output "db_address" {
  value = aws_db_instance.main.address
}
output "db_port" {
  value = aws_db_instance.main.port
}
output "db_name" {
  value = aws_db_instance.main.db_name
}
output "db_secret_arn" {
  value = aws_secretsmanager_secret.db_password.arn
}
