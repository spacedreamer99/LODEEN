terraform {
  required_version = ">= 1.6"
  required_providers {
    clo = {
      source  = "clo-ru/clo"
      version = "~> 2.8.0"
    }
  }
}

provider "clo" {
  auth_url = "https://api.clo.ru"
  token    = var.clo_token
}
