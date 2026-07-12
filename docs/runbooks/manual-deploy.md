# Manual deploy

Rebuild and deploy any ref to production on demand. Unlike the automatic pipeline, which only deploys `main` on push, this deploys any commit that has passed CI, so you can ship a branch without merging. To undo a bad deploy, use [rollback](rollback.md) instead.

## Run

Dispatch [manual-deploy.yml](https://github.com/ALucek/chat-lucek/actions/workflows/manual-deploy.yml) from the Actions tab:

- `ref`: branch, tag, or commit to deploy (default `main`)
- `service`: `api`, `web`, `agent`, or `all`

It resolves the ref to a commit, refuses unless that commit has a passing `test` run, then builds and deploys the selected services (migrations run first for the api).

## Verify

The run confirms the web root returns 200 and `/api/me` returns 401. That does not reach the agent (it is IAM-locked), so after an `agent` deploy, confirm with a real chat message.

## Notes

- The CI gate has no bypass: a commit with no passing `test` run cannot deploy. If production is down and the fix has not passed CI, [roll back](rollback.md) instead.
- Deploying an older commit runs older code against the current schema. Migrations are forward-only and backward-compatible (see [rollback](rollback.md)), so this is safe; a very old commit is a judgment call.
- Manual deploys share the `deploy-production` concurrency group, so they never race an automatic deploy or a rollback.
