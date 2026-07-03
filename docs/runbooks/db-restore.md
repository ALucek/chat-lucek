# Restore the database

Cloud SQL keeps 7 daily automated backups and 7 days of point-in-time history for the `chat` instance. Deletion protection is on. There are two recovery paths: restore a backup in place, or recover to an exact moment by cloning.

## Restore a backup in place

For data loss or corruption. This overwrites the current instance with the backup; the app keeps its connection and reconnects, so nothing needs reconfiguring.

```
# list backups, newest first
gcloud sql backups list --instance=chat

# restore one onto the same instance (destructive)
gcloud sql backups restore <BACKUP_ID> --restore-instance=chat
```

If the backup predates a schema migration, re-apply migrations afterward:

```
gcloud run jobs execute chat-migrate --region=us-central1 --wait
```

## Recover to a point in time

For "undo the last N minutes," or as a safe drill. Cloning is non-destructive: it builds a new instance as of a timestamp, leaving `chat` untouched.

```
gcloud sql instances clone chat chat-restore \
  --point-in-time '2026-07-03T21:00:00Z'
```

Inspect `chat-restore`. To make it the live database you repoint the app's Cloud SQL connection (a Terraform change). For a drill, tear it down when done (clones inherit deletion protection, so clear it first):

```
gcloud sql instances patch chat-restore --no-deletion-protection
gcloud sql instances delete chat-restore
```

## Notes

- Retention: 7 daily backups and 7 days of point-in-time logs. Anything older is unrecoverable.
- Deletion protection is on, so the primary instance cannot be dropped by accident.
