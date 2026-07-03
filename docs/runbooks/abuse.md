# Abuse response

An [LB 429 spike](../monitoring.md) means requests are being rate-limited in bulk: usually abuse, sometimes a limit biting real traffic. Cloud Armor already throttles the edge (10/min on the auth paths, 120/min on `/api/`); this ladder is for when that is not enough. Work from targeted to blunt.

## Assess

Find out who is being throttled before acting. In the Cloud Monitoring console, look at the 429s on the `chat-url-map` load balancer and the API logs for the source IPs and paths. One user tripping a limit is noise; a flood is not.

## Block an IP

If the abuse is a handful of addresses, deny them at the edge:

```
gcloud compute security-policies rules create 100 \
  --security-policy=chat-api-policy \
  --action=deny-403 --src-ip-ranges=<IP-or-CIDR>
```

Remove it when the abuse stops:

```
gcloud compute security-policies rules delete 100 \
  --security-policy=chat-api-policy
```

## Escalate

- Account-creation abuse: [close signups](disable-signup.md).
- A broad flood you cannot pin to IPs: [full-kill](full-kill.md) denies all API traffic at the edge.

## Incident lockdown

If you suspect a compromise rather than just load: [full-kill](full-kill.md), then [rotate the JWT key](rotate-jwt.md) if the signing key may be exposed.
