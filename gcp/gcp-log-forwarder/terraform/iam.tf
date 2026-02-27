# Fetch the project number and ID automatically
data "google_project" "project" {}

# -----------------------------------------------------------------------------
# 1. LOG SINK -> PUBSUB (The Entry)
# -----------------------------------------------------------------------------
resource "google_pubsub_topic_iam_member" "log_sink_publisher" {
  topic      = google_pubsub_topic.log_topic.name
  role       = "roles/pubsub.publisher"
  member     = google_logging_project_sink.sw_sink.writer_identity
  depends_on = [time_sleep.api_wait]
}

# -----------------------------------------------------------------------------
# 2. PUBSUB -> GCS BUCKET (The Buffer)
# -----------------------------------------------------------------------------
resource "google_storage_bucket_iam_member" "pubsub_gcs_writer" {
  bucket     = google_storage_bucket.log_bucket.name
  role       = "roles/storage.objectCreator"
  member     = "serviceAccount:service-${data.google_project.project.number}@gcp-sa-pubsub.iam.gserviceaccount.com"
  depends_on = [time_sleep.api_wait]
}

resource "google_storage_bucket_iam_member" "pubsub_gcs_reader_legacy" {
  bucket     = google_storage_bucket.log_bucket.name
  role       = "roles/storage.legacyBucketReader"
  member     = "serviceAccount:service-${data.google_project.project.number}@gcp-sa-pubsub.iam.gserviceaccount.com"
  depends_on = [time_sleep.api_wait]
}

# -----------------------------------------------------------------------------
# 3. CLOUD BUILD (The Factory)
# -----------------------------------------------------------------------------
resource "google_project_iam_member" "cloudbuild_repo_writer" {
  project = var.project_id
  role    = "roles/artifactregistry.writer"
  member  = "serviceAccount:${data.google_project.project.number}@cloudbuild.gserviceaccount.com"
  depends_on = [time_sleep.api_wait]
}

resource "google_project_iam_member" "compute_builder_repo_writer" {
  project = var.project_id
  role    = "roles/artifactregistry.writer"
  member  = "serviceAccount:${data.google_project.project.number}-compute@developer.gserviceaccount.com"
  depends_on = [time_sleep.api_wait]
}

resource "google_project_iam_member" "cloudbuild_logging" {
  project    = var.project_id
  role       = "roles/logging.logWriter"
  member     = "serviceAccount:${data.google_project.project.number}@cloudbuild.gserviceaccount.com"
  depends_on = [time_sleep.api_wait]
}

# -----------------------------------------------------------------------------
# 4. EVENTARC & GCS TRIGGER (The Bridge)
# -----------------------------------------------------------------------------

resource "google_storage_bucket_iam_member" "eventarc_bucket_viewer" {
  bucket     = google_storage_bucket.log_bucket.name
  role       = "roles/storage.objectViewer"
  member     = "serviceAccount:service-${data.google_project.project.number}@gcp-sa-eventarc.iam.gserviceaccount.com"
  depends_on = [time_sleep.api_wait]
}

# The actual trigger receiver permission
resource "google_project_iam_member" "eventarc_run_invoker" {
  project    = var.project_id
  role       = "roles/run.invoker"
  member     = "serviceAccount:${data.google_project.project.number}-compute@developer.gserviceaccount.com"
  depends_on = [time_sleep.api_wait]
}

# Standard Eventarc Service Agent Roles
resource "google_project_iam_member" "eventarc_service_agent_role" {
  project    = var.project_id
  role       = "roles/eventarc.serviceAgent"
  member     = "serviceAccount:service-${data.google_project.project.number}@gcp-sa-eventarc.iam.gserviceaccount.com"
  depends_on = [google_project_service_identity.eventarc_sa]
}

# Permissions for the GCS Service Account (needed for Eventarc to hear GCS)
resource "google_project_iam_member" "gcs_pubsub_publisher" {
  project    = var.project_id
  role       = "roles/pubsub.publisher"
  member     = "serviceAccount:service-${data.google_project.project.number}@gs-project-accounts.iam.gserviceaccount.com"
  depends_on = [google_project_service_identity.gcs_sa]
}
# -----------------------------------------------------------------------------
# 5. FUNCTION RUNTIME (The Consumer)
# -----------------------------------------------------------------------------
resource "google_project_iam_member" "function_storage_viewer" {
  project    = var.project_id
  role       = "roles/storage.objectViewer"
  member     = "serviceAccount:${data.google_project.project.number}-compute@developer.gserviceaccount.com"
  depends_on = [time_sleep.api_wait]
}

resource "google_project_iam_member" "function_event_receiver" {
  project    = var.project_id
  role       = "roles/eventarc.eventReceiver"
  member     = "serviceAccount:${data.google_project.project.number}-compute@developer.gserviceaccount.com"
  depends_on = [time_sleep.api_wait]
}

# -----------------------------------------------------------------------------
# 6. DETERMINISTIC WAIT: IAM Propagation (60s)
# -----------------------------------------------------------------------------
resource "time_sleep" "iam_wait" {
  depends_on = [
    google_project_iam_member.cloudbuild_repo_writer,
    google_project_iam_member.compute_builder_repo_writer,
    google_project_iam_member.cloudbuild_logging,
    google_storage_bucket_iam_member.eventarc_bucket_viewer,
    google_project_iam_member.eventarc_run_invoker,
    google_project_iam_member.eventarc_service_agent_role,
    google_storage_bucket_iam_member.pubsub_gcs_writer,
    google_storage_bucket_iam_member.pubsub_gcs_reader_legacy,
    google_pubsub_topic_iam_member.log_sink_publisher,
    google_project_iam_member.function_storage_viewer,
    google_project_iam_member.function_event_receiver
  ]

  create_duration = "60s"
}