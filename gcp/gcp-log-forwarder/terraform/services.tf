locals {
  services = [
    "artifactregistry.googleapis.com",
    "cloudbuild.googleapis.com",
    "cloudfunctions.googleapis.com",
    "run.googleapis.com",
    "logging.googleapis.com",
    "pubsub.googleapis.com",
    "eventarc.googleapis.com",
    "storage.googleapis.com"
  ]
}

resource "google_project_service" "apis" {
  for_each           = toset(local.services)
  project            = var.project_id
  service            = each.key
  disable_on_destroy = false
}

# The deterministic anchor: Everything else waits for this.
resource "time_sleep" "api_wait" {
  depends_on      = [google_project_service.apis]
  create_duration = "60s"
}

resource "google_project_service_identity" "gcs_sa" {
  provider = google-beta
  project  = var.project_id
  service  = "storage.googleapis.com"
}

# Forces the creation of the Eventarc Service Agent identity
resource "google_project_service_identity" "eventarc_sa" {
  provider = google-beta
  project  = var.project_id
  service  = "eventarc.googleapis.com"
}