variable "clo_token" {
  description = "API token for CLO (FirstVDS)"
  type        = string
  sensitive   = true
}

variable "server_name" {
  description = "Server name"
  type        = string
  default     = "lodeen-prod-1"
}

variable "server_plan" {
  description = "Server plan (e.g., r1000, r2000)"
  type        = string
  default     = "r1000"
}
