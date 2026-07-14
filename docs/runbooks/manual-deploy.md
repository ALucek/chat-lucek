# Manual deploy

Rebuild and deploy any ref straight to production on demand, bypassing the release train. The automatic pipeline deploys `main` to dev on push and promotes to prod on a cadence; this ships a chosen service to prod immediately, so you can hotfix a branch without merging or waiting for the next promotion. It deploys any commit that has passed CI. To undo a bad deploy, use [rollback](rollback.md) instead.

## Run

Dispatch [manual-deploy.yml](https://github.com/ALucek/chat-lucek/actions/workflows/manual-deploy.yml) from the Actions tab:

- `ref`: branch, tag, or commit to deploy (default `main`)
- `service`: `api`, `web`, `agent`, or `all`

It resolves the ref to a commit, refuses unless that commit has a passing `test` run, then builds and deploys the selected services to dev (migrations run first for the api), verifies them, and promotes them straight to production.

## Verify

The run verifies each service on the dev host before promoting (api `/readyz` + `/api/me`, web homepage, the agent via the api's `/agentz` integration check), then confirms the production domain after the flip. No manual check is needed.

## Notes

- The CI gate has no bypass: a commit with no passing `test` run cannot deploy. If production is down and the fix has not passed CI, [roll back](rollback.md) instead.
- Deploying an older commit runs older code against the current schema. Migrations are forward-only and backward-compatible (see [rollback](rollback.md)), so this is safe; a very old commit is a judgment call.
- Manual deploys share the `deploy-production` concurrency group with the automatic deploy, the scheduled promote, and rollback, so they never race one another.
