# Maintenance mode

Serve a friendly maintenance page while the API returns 503, for planned downtime or a risky migration. Health endpoints stay up, so the uptime alerts do not fire.

## Run

Dispatch [maintenance.yml](https://github.com/ALucek/chat-lucek/actions/workflows/maintenance.yml) from the Actions tab with `state`:

- `on`: the web serves the maintenance page and the API returns 503 on every route except health
- `off`: back to normal

It sets `MAINTENANCE_MODE` on both Cloud Run services and confirms the resulting state.

## Verify

- On: the web root shows "Down for maintenance" (200), `/api/me` returns 503, `/readyz` returns 200
- Off: web root 200, `/api/me` 401

## Notes

- Transient: do not run `terraform apply` while maintenance is on, or it lifts the flag. Turn it off first.
- An API 5xx alert email during the window is expected (the 503s count as 5xx) and benign.
- The legal pages (`/terms`, `/privacy`) stay reachable so the page's links work.
