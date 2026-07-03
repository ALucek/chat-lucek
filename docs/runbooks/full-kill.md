# Full-kill

Deny all API traffic at the Cloud Armor edge, so a flood or abuse spike never reaches the API (no compute, no LLM calls, no database load). The blunt emergency lever, above [maintenance](maintenance.md) on the ladder. The web frontend keeps serving.

## Run

Dispatch [full-kill.yml](https://github.com/ALucek/chat-lucek/actions/workflows/full-kill.yml) from the Actions tab:

- `state`: `on` adds a top-priority deny rule to the API's Armor policy; `off` removes it
- `confirm`: type `KILL` (guards against an accidental run)

## Verify

The run polls until it converges (Cloud Armor changes take a few minutes to propagate at the edge):

- On: `/api/me` returns 403, web root still 200
- Off: `/api/me` back to 401

## Bring it back up

Dispatch again with `state=off` and `confirm=KILL`. If the workflow itself is unavailable, delete the rule directly (this is a control-plane action, so it works even while the site is denied):

```
gcloud compute security-policies rules delete 1 \
  --security-policy=chat-api-policy
```

## Notes

- The web frontend is not blocked; pair with [maintenance](maintenance.md) if you also want users to see a page.
- This trips the "Site down" alert, since `/readyz` routes through the denied API. Expected during an intentional kill.
