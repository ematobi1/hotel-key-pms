variable "environment" { type = string }
variable "alb_arn" {
  description = "ARN of the ALB to associate the WAF ACL with"
  type        = string
  default     = ""
}
