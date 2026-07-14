# Dev environment

`dev.chat.lucek.ai` serves the candidate (`cand`-tagged) Cloud Run revisions through the same load balancer as production, behind Identity-Aware Proxy. The blue-green deploy verifies a candidate here before promoting it.

## Access

Browsing prompts a Google login. Only principals with `roles/iap.httpsResourceAccessor` on the candidate backends get in (the `owner_email` account and the deploy service account); everyone else gets a 403. CI reaches it with an OIDC token minted for the deploy service account, audience set to the IAP OAuth client ID.

## What it serves

The `cand` tag points at the most recent candidate revision. On a passing deploy the promoted revision keeps the tag, so between deploys the dev host mirrors production. A failed deploy leaves the dev host on the broken candidate while production stays on the last good revision; this is fix-forward, so the next deploy moves the tag and the dev host self-heals. Production is never affected either way.

## Agent

The agent is not served on the dev host; it is internal, called by the api. Its candidate is gated by its `/healthz` startup probe during a deploy, so a broken agent revision never becomes ready and is never promoted. There is no unauthenticated endpoint to smoke, so confirm a new agent release functionally with a real chat message after it promotes.

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
