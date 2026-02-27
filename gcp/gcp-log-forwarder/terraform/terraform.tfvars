project_id    = "SET_PROJECT_ID" // Only set this if an active project was not selected in the previous step.
region        = "SET_REGION" // e.g. us-central1
bucket_name  = "SET_BUCKET_NAME" # Must be globally unique

# SolarWinds Observability Configuration
# Standard OTLP/HTTP endpoint for North America (na-01)
otlp_endpoint = "https://otel.collector.na-01.cloud.solarwinds.com:443:443/v1/logs"

# Replace with your actual Ingestion/API Token
api_token  = ""

topic_name    = "SET_TOPIC_NAME" // e.g. swo-logs-topic
sink_name     = "SET_SINK_NAME" // e.g. swo-logs-sink
function_name = "SET_FUNCTION_NAME" // e.g. swo-logs-function

