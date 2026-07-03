# Rotate secrets

The JWT signing key has its own one-click ([rotate JWT](rotate-jwt.md)). The other three secrets are rotated by hand, since each touches an external system or the live database. The shape is the same: put the new value in Secret Manager, then restart chat-api onto that version.

## Database password

Coordinated: the Postgres user password and the secret change together, and chat-api restarts to pick it up. Expect brief database connection errors between the password change and the restart, so run these back-to-back.

```
pw=$(openssl rand -base64 32)
gcloud sql users set-password app --instance=chat --password="$pw"
ver=$(printf '%s' "$pw" | gcloud secrets versions add db-password --data-file=- --format='value(name)')
gcloud run services update chat-api --region=us-central1 \
  --update-secrets "DB_PASSWORD=db-password:${ver##*/}"
```

## OpenRouter API key

Create a new key in the OpenRouter dashboard, store it, restart, then revoke the old key.

```
ver=$(printf '%s' '<new-key>' | gcloud secrets versions add openrouter-api-key --data-file=- --format='value(name)')
gcloud run services update chat-api --region=us-central1 \
  --update-secrets "OPENROUTER_API_KEY=openrouter-api-key:${ver##*/}"
```

## Google client secret

Rotate the OAuth client secret in the Google Cloud console (APIs and Services, Credentials), then store it and restart.

```
ver=$(printf '%s' '<new-secret>' | gcloud secrets versions add google-client-secret --data-file=- --format='value(name)')
gcloud run services update chat-api --region=us-central1 \
  --update-secrets "GOOGLE_CLIENT_SECRET=google-client-secret:${ver##*/}"
```

## Notes

- Only chat-api reads these; a rotation just cycles its revision.
- Old secret versions stay enabled but unused; disable them once the new revision is healthy.
