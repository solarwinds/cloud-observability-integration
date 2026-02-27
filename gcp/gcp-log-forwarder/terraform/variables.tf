# terraform/variables.tf

variable "project_id" {
  description = "The GCP Project ID where resources will be deployed"
  type        = string
}

variable "region" {
  description = "The GCP region for the bucket and function"
  type        = string
  default     = "us-central1"
}

variable "topic_name" {
  description = "The name of the Pub/Sub topic for log ingestion"
  type        = string
  default     = "swo-logs-topic"
}

variable "sink_name" {
  description = "The name of the Logging Sink"
  type        = string
  default     = "swo-logs-sink"
}

variable "function_name" {
  description = "The name of the Cloud Function forwarder"
  type        = string
  default     = "swo-logs-function"
}

variable "bucket_name" {
  description = "The globally unique name for the log staging bucket"
  type        = string
}

variable "api_token" {
  description = "SolarWinds Observability API Token"
  type        = string
  sensitive   = true
}

variable "otlp_endpoint" {
  description = "SolarWinds OTLP HTTP endpoint"
  type        = string
}