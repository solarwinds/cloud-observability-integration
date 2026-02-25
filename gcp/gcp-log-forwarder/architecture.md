Architecture Design for GCP Log Forwarder
This document outlines the event-driven batch-processed architecture for forwarding Google Cloud Platform logs to SolarWinds Observability (SWO) using OTLP-JSON over HTTPS.

Data Flow
The solution follows a Buffered Pipeline pattern to ensure reliability and cost-efficiency.

Ingestion: The google_logging_project_sink captures logs based on a specific filter and routes them to a Pub/Sub topic.

Buffering: Instead of triggering a function for every single log entry, Pub/Sub batches data into Cloud Storage (GCS).

Batch Triggers: Every 50MB of logs or every 10 minutes, a new .json file is finalized in GCS.

Processing: Eventarc detects the object.v1.finalized event and triggers the Go-based forwarder.

Forwarding: The function reads the GCS object, transforms the logs into OTLP (OpenTelemetry Line Protocol) format, GZIPs the payload, and sends it via a POST request to the SolarWinds HTTPS endpoint.

Key Architectural Decisions
1. Batching vs Streaming
Decision: We use GCS as a buffer instead of a direct Pub/Sub-to-Function trigger.

Why: Direct streaming can lead to thousands of function cold-starts during log spikes. GCS batching allows the function to process thousands of logs in a single execution, significantly reducing Cloud Function costs and preventing API rate-limiting at the SolarWinds ingestion point.

2. OTLP Transformation
Decision: The forwarder maps GCP LogEntry fields to OTLP LogRecords.

Why: SolarWinds native OTLP ingestion requires specific resource attributes and severity mappings. By transforming in-flight, we ensure logs are instantly compatible with SWO dashboards without post-processing.

3. Least-Privilege IAM
Decision: Service agents are restricted to artifactregistry.writer and storage.objectViewer.

4. Deterministic Deployment
Decision: Implementation of time_sleep buffers between API enablement and IAM assignment.

Why: GCP's IAM is eventually consistent. Without these buffers, initial Terraform applies often fail with Service Account Not Found errors.

Cost Optimization and Cleanup
GCS Lifecycle Rules: All log batches are automatically deleted after a short retention period (2 days) using GCS lifecycle management.

Network Egress: The function is set to ALLOW_INTERNAL_ONLY for the trigger, and the payload is GZIP compressed to minimize data transfer costs to the SolarWinds endpoint.

Memory Tuning: Configured to balance the memory-intensive sync.Pool usage during GZIP compression with execution cost.