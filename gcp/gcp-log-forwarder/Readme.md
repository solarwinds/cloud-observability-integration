# GCP to SolarWinds Log Forwarder

This repository contains the Terraform configuration to deploy an event-driven logging pipeline from Google Cloud to SolarWinds Observability.

### 📋 Pre-requisites
* **Terraform (v1.5+)**
* **Google Cloud SDK (gcloud)**
* **Project ID** of the target GCP project.
* **SolarWinds API Token** and OTLP Endpoint.

---

##  Step 1: Project-Specific Authentication
Run these commands to lock your terminal session to the specific project. This ensures Terraform uses the correct quotas and permissions for the target project.

```bash
# 1. Set the CLI context
gcloud config set project [PROJECT_ID]

# 2. Authenticate Terraform (ADC) specifically for this project
gcloud auth application-default login
```

---

## Step 2: Terraform Execution

```bash
# 1. Initialize (Run once)
terraform init --reconfigure

terraform plan

terraform apply
```

---

##  Cleanup
To tear down the infrastructure for a specific project while staying in the correct workspace:

```bash
terraform destroy
```