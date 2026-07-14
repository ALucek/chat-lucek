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

## After a rollback

Rollback moves both production traffic and the dev (`cand`) tag onto the target revision, so dev matches prod and the next scheduled [promotion](../deployment.md) is a no-op instead of re-shipping the bad revision. To move forward, push a fix to `main`: it deploys to dev, and the next promotion ships it. There is no separate resume step.

## Notes

- Migrations are additive and backward-compatible, so rolling the image back is safe against the live schema. To undo a schema change, ship a new forward migration.
- Rollback shifts traffic only; it never runs a migration.
