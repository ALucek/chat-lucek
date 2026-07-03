# Disable signup

Open or close new-user registration. Existing users are unaffected. Terraform owns the flag, so there is no drift and it can stay closed indefinitely.

## Run

Set `signup_open` in [`infra/terraform.tfvars`](../../infra/terraform.tfvars) and apply:

```
signup_open = false   # true to reopen
```

```
cd infra && terraform apply
```

The apply rolls chat-api to a new revision with the flag set. New sign-ins then get a 403; existing sessions keep working.

## Verify

A clean plan means the flag is applied:

```
cd infra && terraform plan
```

When closed, a brand-new Google account is refused at sign-in.

## Notes

- This is a deliberate, durable toggle, which is why it is Terraform rather than a one-click workflow.
