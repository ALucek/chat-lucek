terraform {
  required_version = ">= 1.9"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
    langsmith = {
      source  = "langchain-ai/langsmith"
      version = "~> 0.0.2"
    }
  }
}