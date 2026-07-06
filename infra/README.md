## Setup

Requires Terraform 1.9+ and the gcloud CLI. Authenticate, configure, and initialize from `infra/`:

```bash
gcloud auth application-default login
cp backend.hcl.example backend.hcl         # your Terraform state bucket
cp infra.tfvars.example terraform.tfvars   # project_id, domain, etc.
terraform init -backend-config=backend.hcl
terraform plan
```

Full first-time deployment (state bucket, secret values, DNS) lives in the [deployment guide](../docs/deployment.md).

## Resources

| File | Provisions |
| --- | --- |
| `apis.tf` | Enables the required GCP APIs |
| `cloud_run.tf` | `web`, `api`, and `agent` services, the migration job, service accounts + IAM |
| `cloud_sql.tf` | Postgres instance, database, and app user |
| `lb.tf` | Global HTTPS load balancer, managed cert, HTTP-to-HTTPS redirect |
| `armor.tf` | Cloud Armor edge policy |
| `artifact_registry.tf` | Docker image repository |
| `secrets.tf` | Secret Manager secrets |
| `deploy.tf` | Workload Identity Federation + deploy service account for CI |
| `monitoring.tf` | Uptime checks, alert policies, and log metrics |
| `budget.tf` | Monthly billing budget alert |
| `langsmith.tf` | Online eval evaluators and run rules |
