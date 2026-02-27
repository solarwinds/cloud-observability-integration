# 1. Source Code Packaging
# Automatically zips the /src folder whenever a file changes
data "archive_file" "source" {
  type        = "zip"
  source_dir  = "${path.module}/../src"
  output_path = "${path.module}/function-source.zip"
}

# 2. Artifact Registry (Stores the Function container image)
resource "google_artifact_registry_repository" "repo" {
  location      = var.region
  repository_id = "gcp-log-forwarder-repo"
  format        = "DOCKER"
  # Mandatory wait for APIs to wake up
  depends_on    = [time_sleep.api_wait]
}

# 3. Storage Bucket (Staging area for logs and source code)
resource "google_storage_bucket" "log_bucket" {
  name                        = var.bucket_name
  location                    = var.region
  uniform_bucket_level_access = true
  force_destroy               = true

  lifecycle_rule {
    condition {
      age = 2
      matches_prefix = ["sw-batches/"] # Only delete the log files
    }
    action {
      type = "Delete"
    }
  }

  depends_on                  = [time_sleep.api_wait]
}

# 4. Upload zipped source code to GCS
resource "google_storage_bucket_object" "source_zip" {
  name   = "deployments/src-${data.archive_file.source.output_md5}.zip"
  bucket = google_storage_bucket.log_bucket.name
  source = data.archive_file.source.output_path
}

# 5. Pub/Sub Topic (Incoming log buffer)
resource "google_pubsub_topic" "log_topic" {
  name       = var.topic_name
  depends_on = [time_sleep.api_wait]
}

# 6. GCS Sink Subscription (Batches data: 50MB or 10 Min)
resource "google_pubsub_subscription" "gcs_sink" {
  name  = "gcp-logs-gcs-sink-sub"
  topic = google_pubsub_topic.log_topic.name
  cloud_storage_config {
    bucket       = google_storage_bucket.log_bucket.name
    filename_prefix  = "sw-batches/"
    filename_suffix  = ".json"
    max_duration = "600s"
    max_bytes    = 52428800
  }
  # Ensures bucket and topic exist before linking them
  depends_on = [
    google_storage_bucket.log_bucket,
    google_pubsub_topic.log_topic,
    google_storage_bucket_iam_member.pubsub_gcs_writer,
    google_storage_bucket_iam_member.pubsub_gcs_reader_legacy
  ]
}

# 7. Log Sink (The filter to exclude Cloud Run/Function noise)
resource "google_logging_project_sink" "sw_sink" {
  name        = var.sink_name
  destination = "pubsub.googleapis.com/${google_pubsub_topic.log_topic.id}"
  filter      = "severity >= INFO AND NOT (resource.type:\"cloud_run\" OR resource.type=\"cloud_function\" OR logName:\"otelscope\" OR resource.type=\"dns_query\")"
  unique_writer_identity = true
}

# 8. The Forwarder (Cloud Function Gen 2)
resource "google_cloudfunctions2_function" "gcp_forwarder" {
  name     = var.function_name
  location = var.region

  build_config {
    runtime           = "go125"
    entry_point       = "HandleGcsBatch"
    docker_repository = google_artifact_registry_repository.repo.id
    source {
      storage_source {
        bucket = google_storage_bucket.log_bucket.name
        object = google_storage_bucket_object.source_zip.name
      }
    }
  }

  service_config {
    max_instance_count = 10
    available_memory   = "512Mi"
    timeout_seconds    = 120
    ingress_settings = "ALLOW_INTERNAL_ONLY"
    environment_variables = {
      SWI_API_KEY       = var.api_token
      SWI_OTEL_ENDPOINT = var.otlp_endpoint
    }
  }

  event_trigger {
    trigger_region = var.region
    event_type     = "google.cloud.storage.object.v1.finalized"
    event_filters {
      attribute = "bucket"
      value     = google_storage_bucket.log_bucket.name
    }
  }

  # FINAL DETERMINISTIC LOCK
  # Only deploy once IAM is live and the code object is uploaded
  depends_on = [time_sleep.iam_wait, google_storage_bucket_object.source_zip]
}