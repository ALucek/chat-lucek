# Rotate secrets

The JWT signing key has its own one-click ([rotate JWT](rotate-jwt.md)). The rest are rotated by hand, since each touches an external system or the live database. The shape is the same: put the new value in Secret Manager, then restart the service that reads it. chat-api reads `db-password` and `google-client-secret`; chat-agent reads `openrouter-api-key`, `tavily-api-key`, and `langsmith-api-key`.

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

## Notes

- A rotation just cycles the reading service's revision (chat-api or chat-agent).
- Old secret versions stay enabled but unused; disable them once the new revision is healthy.
