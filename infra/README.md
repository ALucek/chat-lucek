## Setup

Requires Terraform 1.9+ and the gcloud CLI. Authenticate, configure, and initialize from `infra/`:

```bash
gcloud auth application-default login
cp backend.hcl.example backend.hcl         # your Terraform state bucket
cp terraform.tfvars.example terraform.tfvars   # project_id, domain, etc.
terraform init -backend-config=backend.hcl
terraform plan
```

Full first-time deployment (state bucket, secret values, DNS) lives in the [deployment guide](../docs/deployment.md).

## Variables

Set in `terraform.tfvars`. Required vars have no default; the rest are shown with their defaults in `terraform.tfvars.example`.

| Variable | Default | Purpose |
| --- | --- | --- |
| `project_id` | required | GCP project hosting the deployment |
| `google_client_id` | required | Google OAuth client ID the API verifies tokens against |
| `owner_email` | required | Account granted the unlimited run budget |
| `billing_account` | required | Billing account ID for the budget |
| `langsmith_api_key` | required | LangSmith key for provisioning online evaluators |
| `langsmith_workspace_id` | required | LangSmith workspace (tenant) that owns the evaluators |
| `injection_prompt_commit` | required | Pinned Prompt Hub commit for the prompt-injection judge |
| `thread_helpfulness_prompt_commit` | required | Pinned Prompt Hub commit for the thread-helpfulness judge |
| `resend_api_key` | required | Resend send-scoped key for alerts and sign-in email |
| `region` | `us-central1` | Region for regional resources |
| `domain` | `chat.lucek.ai` | Public domain served by the load balancer |
| `db_tier` | `db-f1-micro` | Cloud SQL machine tier |
| `github_repo` | `ALucek/chat-lucek` | Repo allowed to deploy via Workload Identity |
| `signup_open` | `false` | Whether new-user registration is allowed |
| `magic_link_from` | `login@lucek.ai` | Verified Resend sender for sign-in emails |
| `budget_amount` | `20` | Monthly budget in USD; thresholds alert as spend crosses it |
| `alert_email_from` | `alerts@lucek.ai` | From address for alert emails; must be a verified Resend sender |
| `langsmith_project` | `chat-lucek-ai-prod` | LangSmith project the agent traces into |
| `langsmith_project_dev` | `chat-lucek-dev` | LangSmith project for dev-host runs, off the prod evaluators |
| `agent_default_model` | `deepseek/deepseek-v4-flash` | Default model the agent uses |
| `agent_max_searches` | `5` | Max web searches per agent run |

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
