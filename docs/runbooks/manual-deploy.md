# Manual deploy

Rebuild and deploy a chosen ref to production on demand. The automatic pipeline only deploys `main` after a push; this ships any commit that has passed CI, so you can deploy a branch without merging. Use it to preview a branch in production, redeploy after a deploy failed past the build, or force a clean rebuild. To undo a bad deploy, use [rollback](rollback.md) instead.

## Run

Dispatch [manual-deploy.yml](https://github.com/ALucek/chat-lucek/actions/workflows/manual-deploy.yml) from the Actions tab:

- `ref`: branch, tag, or commit to deploy (default `main`)
- `service`: `api`, `web`, `agent`, or `all`

It resolves the ref to a commit and refuses unless that commit has a successful `test` run, then builds and deploys the selected services exactly as the automatic pipeline does: build and push the image, run migrations for the api, update the Cloud Run service, and smoke-check the site.

## Verify

The run confirms the web root returns 200 and `/api/me` returns 401. That smoke does not exercise the agent (it is IAM-locked and not reachable unauthenticated), so after an `agent` deploy, confirm with a real chat message. To see the serving revision of any service:

```
gcloud run services describe chat-api --region=us-central1 \
  --format='value(status.traffic[].revisionName)'
```

## Notes

- The CI gate is absolute. A commit with no successful `test` run cannot deploy, and there is no bypass. If production is down and the fix has not passed CI, [roll back](rollback.md) instead.
- Deploying an older commit runs older code against the current schema. Migrations are forward-only and backward-compatible (see [rollback](rollback.md)), so an older image is safe, but deploying a very old commit is a judgment call.
- Manual deploys share the `deploy-production` concurrency group, so one never races an automatic deploy or a rollback.
