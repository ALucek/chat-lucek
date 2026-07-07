# Rotate secrets

The JWT signing key has its own one-click ([rotate JWT](rotate-jwt.md)). The rest are rotated by hand, since each touches an external system or the live database. For the Cloud Run services the shape is the same: put the new value in Secret Manager, then restart the service that reads it. chat-api reads `db-password` and `google-client-secret`; chat-agent reads `openrouter-api-key`, `tavily-api-key`, and `langsmith-api-key`. Some keys also live outside Secret Manager, in Terraform vars or GitHub Actions secrets, so rotating one means replacing every copy (see [Keys outside Secret Manager](#keys-outside-secret-manager)).

## Database password

Coordinated: the Postgres user password and the secret change together, and chat-api restarts to pick it up. Expect brief database connection errors between the password change and the restart, so run these back-to-back.

```
pw=$(openssl rand -base64 32)
gcloud sql users set-password app --instance=chat --password="$pw"
ver=$(printf '%s' "$pw" | gcloud secrets versions add db-password --data-file=- --format='value(name)')
gcloud run services update chat-api --region=us-central1 \
  --update-secrets "DB_PASSWORD=db-password:${ver##*/}"
```

## Google client secret

Rotate the OAuth client secret in the Google Cloud console (APIs and Services, Credentials), then store it and restart.

```
ver=$(printf '%s' '<new-secret>' | gcloud secrets versions add google-client-secret --data-file=- --format='value(name)')
gcloud run services update chat-api --region=us-central1 \
  --update-secrets "GOOGLE_CLIENT_SECRET=google-client-secret:${ver##*/}"
```

## Agent provider keys

The agent reads three provider keys. Rotate each at its dashboard (OpenRouter, Tavily, LangSmith), store the new value, and restart chat-agent. OpenRouter and Tavily serve live traffic, so revoke the old key only once the new revision is healthy.

```
# <secret> = openrouter-api-key | tavily-api-key | langsmith-api-key
# <ENV>    = OPENROUTER_API_KEY | TAVILY_API_KEY | LANGSMITH_API_KEY
ver=$(printf '%s' '<new-key>' | gcloud secrets versions add <secret> --data-file=- --format='value(name)')
gcloud run services update chat-agent --region=us-central1 \
  --update-secrets "<ENV>=<secret>:${ver##*/}"
```

## Keys outside Secret Manager

The LangSmith online-eval infra ([langsmith.tf](../../infra/langsmith.tf)) and the weekly [evals workflow](../../.github/workflows/evals.yml) keep their own copies of some keys, in Terraform vars (`infra/infra.tfvars`) and GitHub Actions secrets. A provider key can live in several places at once, and rotating it means replacing every copy.

| Key | Where it lives |
| --- | --- |
| LangSmith | `langsmith-api-key` (Secret Manager), `langsmith_api_key` (tfvars), `LANGSMITH_API_KEY` (Actions) |
| OpenRouter | `openrouter-api-key` (Secret Manager), `OPENROUTER_API_KEY` (Actions) |
| Tavily | `tavily-api-key` (Secret Manager), `TAVILY_API_KEY` (Actions) |
| Resend | `resend_api_key` (tfvars), `RESEND_API_KEY` (Actions) |

Rotate the Secret Manager copies as above. For the rest, create the new key at its dashboard, update both homes, then revoke the old one. None of these serve live traffic, so ordering is not sensitive.

```
# tfvars: edit infra/infra.tfvars, then re-apply (langsmith.tf reads these)
cd infra && terraform apply

# GitHub Actions secret
gh secret set <NAME> --body '<new-key>'
```

`EVAL_REPORT_TO` is the one Actions secret that is not a credential: it is the owner email the report goes to, kept as a secret so it is not printed publicly.

## Notes

- A rotation just cycles the reading service's revision (chat-api or chat-agent).
- Old secret versions stay enabled but unused; disable them once the new revision is healthy.
