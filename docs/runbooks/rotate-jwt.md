# Rotate JWT

Rotate the JWT signing key to invalidate every live and forged access token at once. Use it if the key may be compromised. Signed-in users re-key transparently through their refresh token, so no one is logged out.

## Run

Dispatch [rotate-jwt.yml](https://github.com/ALucek/chat-lucek/actions/workflows/rotate-jwt.yml) from the Actions tab. No inputs.

It generates a fresh secret, adds a `jwt-secret` version, restarts chat-api onto it, then checks the site is responding.

## Verify

The run restarts chat-api onto the new version and the site stays healthy (web root 200, `/api/me` 401). Confirm a new enabled version is on top:

```
gcloud secrets versions list jwt-secret --format='table(name,state)'
```

## Notes

- This invalidates access tokens, not sessions: signed-in users get a fresh token from their refresh token without re-logging in.
- Older versions stay enabled in Secret Manager but unused; disable them if you like.
