# Dev environment

`dev.chat.lucek.ai` serves the `cand`-tagged Cloud Run revisions through the same load balancer as production, behind Identity-Aware Proxy. It always reflects the latest `main`: every push deploys here, and production is promoted from it on a cadence.

## Access

Browsing prompts a Google login. Only principals with `roles/iap.httpsResourceAccessor` on the candidate backends get in (the `owner_email` account and the deploy service account); everyone else gets a 403. CI reaches it with an OIDC token minted for the deploy service account, audience set to the IAP OAuth client ID.

## What it serves

The `cand` tag points at the most recent deployed revision, so the dev host always runs the latest `main`. Between promotions dev is ahead of production; a promotion flips prod to whatever dev is serving, after which prod matches dev until the next push. A failed deploy leaves the dev host on the broken revision while production stays on the last promoted revision; this is fix-forward, so the next deploy moves the tag and the dev host self-heals. Production is never affected by a dev deploy.

## Release to prod

Production is promoted from dev every two hours by [promote.yml](../../.github/workflows/promote.yml), and on demand: dispatch it from the Actions tab to release the current dev snapshot immediately. A promotion flips production traffic to the revisions dev is serving, across all three services, and is a no-op when dev already matches prod. To ship one service straight to prod outside the train, use [manual deploy](manual-deploy.md). To undo a bad release, [roll back](rollback.md).

## Agent

The agent is not served on the dev host; it is internal, called by the api. Its revision is checked two ways when it deploys to dev: its `/healthz` startup probe, and a live inference smoke that posts a real chat to the revision and requires a streamed answer with non-zero token usage. The same live smoke re-runs at promotion, catching external drift such as an expired model key, so a broken agent is caught before it serves a user.

## Reset the dev host

To point the dev host back at the current serving revision without a redeploy (for example, to clear a broken candidate), move the `cand` tag onto the serving revision:

```
gcloud run services update-traffic chat-api --region us-central1 \
  --update-tags cand=$(gcloud run services describe chat-api --region us-central1 \
  --format='value(status.traffic[0].revisionName)')
```

Repeat for `chat-web`. This only moves the `cand` tag; it never shifts production traffic.

## Notes

- IAP needs its service agent provisioned once (`gcloud beta services identity create --service=iap.googleapis.com`); see [deployment](../deployment.md).
- The dev host and production share one load balancer and Cloud Armor policy; each host has its own managed certificate.
