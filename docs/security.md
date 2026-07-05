# Security

Defense in depth across the application, infrastructure, and supply chain, backed by automated scanning on every change.

## Scanning

Run the scanners locally:

```bash
make security         # govulncheck, gosec, pip-audit, bandit, pnpm audit, gitleaks
make scan-images      # Trivy scan of the built container images
make tf-config-scan   # Trivy scan of the Terraform config
```

| Tool | Scope |
| --- | --- |
| govulncheck | Go dependency vulnerabilities |
| gosec | Go static analysis |
| pip-audit | Python dependency vulnerabilities |
| bandit | Python static analysis |
| pnpm audit | npm dependency vulnerabilities |
| gitleaks | Committed secrets |
| Trivy | Container images and Terraform config |

CI runs these in [security.yml](../.github/workflows/security.yml) on every push, every pull request, and weekly. Dependabot opens weekly update PRs for Go modules, npm packages, Python packages, GitHub Actions, and Docker base images.

## Application

- **Authentication:** Google Sign-In is the only login path. The API issues a short-lived HS256 access token whose algorithm is pinned on verify, rejecting forgeries like `alg: none`.
- **Refresh tokens:** stored hashed, delivered in an httpOnly, Secure, SameSite=Strict cookie scoped to `/api`. Each refresh rotates the token within a family; reusing a revoked token revokes the whole family as suspected theft.
- **Origin checks:** state-changing requests (POST, PATCH, DELETE) with a mismatched Origin are refused.
- **Security headers:** CSP, HSTS, X-Frame-Options, X-Content-Type-Options, Referrer-Policy, and Permissions-Policy on every response.
- **Limits:** request bodies are capped at 1 MiB, chat sends are rate-limited per user, and each user has a daily run budget.

## Infrastructure

- **HTTPS only:** a Google-managed certificate terminates TLS, and plain HTTP redirects to HTTPS.
- **Edge protection:** a Cloud Armor policy fronts the load balancer.
- **Secrets:** every secret lives in Secret Manager, never in the repo or an image.
- **Deploy identity:** GitHub Actions authenticate through Workload Identity Federation, so there are no long-lived service account keys.
- **Least privilege:** each Cloud Run service runs as its own service account with only the roles it needs.
- **Internal service:** the agent grants its invoke role only to the API's service account, which calls it with a Google-signed ID token; it is not publicly invocable.

## Supply chain

- **Pinned Actions:** every GitHub Action is pinned to a commit SHA.
- **Attestations:** each deploy publishes an SBOM and a build-provenance attestation for the api, web, and agent images.
- **Dependabot:** weekly dependency-update PRs.

## Repository

- **Branch protection:** `main` requires passing status checks (tests, e2e, and the security scans) and cannot be bypassed, including by admins.
- **CODEOWNERS:** changes request review from the repo owner.
- **Squash merges only:** each merge to `main` lands as a single commit.
