terraform {
  required_version = ">= 1.10"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
  }
}

provider "aws" {
  region = var.regiao
  default_tags {
    tags = {
      Projeto = "lisura-poc"
      Dados   = "sinteticos"
      Gerido  = "terraform"
    }
  }
}

data "aws_caller_identity" "conta" {}
data "aws_region" "atual" {}
