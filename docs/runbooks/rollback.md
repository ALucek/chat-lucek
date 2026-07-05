# Rollback

Shift Cloud Run traffic back to a prior revision when a deploy breaks the API or web. No rebuild, near-instant.

## Run

Dispatch [rollback.yml](https://github.com/ALucek/chat-lucek/actions/workflows/rollback.yml) from the Actions tab:

- `service`: `api`, `web`, `agent`, or `all`
- `revision`: blank rolls back to the revision serving before the current one; a name targets that revision

It logs the revision list and the before/after serving revision, then checks the site is responding.

## Verify

The run confirms the web root returns 200 and `/api/me` returns 401. That smoke does not exercise the agent (it is IAM-locked and not reachable unauthenticated), so after an `agent` rollback, confirm with a real chat message. To see the serving revision of any service:

```
gcloud run services describe chat-agent --region=us-central1 \
  --format='value(status.traffic[].revisionName)'
```

## Resume normal deploys

A rollback pins traffic to a revision, so later CD deploys stop serving. Once the fix is deployed, dispatch the workflow again with `revision=latest` to restore automatic routing.

## Notes

- Migrations are additive and backward-compatible, so rolling the image back is safe against the live schema. To undo a schema change, ship a new forward migration.
- Rollback shifts traffic only; it never runs a migration.
