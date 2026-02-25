terraform {
  required_version = ">= 1.5.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0" # Ensures we have the latest Gen2 Function features
    }
    archive = {
      source  = "hashicorp/archive"
      version = "~> 2.4" # Used for the Auto-Zip logic
    }
    time = {
      source  = "hashicorp/time"
      version = "~> 0.11"
    }
    google-beta = {
      source  = "hashicorp/google-beta"
      version = "~> 5.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

provider "google-beta" {
  project = var.project_id
  region  = var.region
}