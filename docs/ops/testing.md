# Testing Policy

`terraform-provider-zohomail` currently uses a **local-first** testing model.

## Default Local Gate

Run these commands for normal provider work:

```bash
make test
make build
make generate
go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs validate
```

Use `make fmt` before committing code changes.

This is the default repo gate. There is no GitHub Actions CI gate yet.

## When Acceptance Is Required

Run `make testacc` when a change touches any of these areas:

- Zoho Mail API request or response mapping
- Terraform resource lifecycle behavior
- import IDs or import state handling
- idempotence or refresh behavior
- remote drift detection

Pure README/runbook changes do not require acceptance.

## Acceptance Environments

Base provider acceptance:

- `TF_ACC=1`
- `ZOHOMAIL_ACCESS_TOKEN`
- `ZOHOMAIL_DATA_CENTER`
- `ZOHOMAIL_ORGANIZATION_ID`

Any scenario that creates disposable Zoho domains or mailboxes also needs:

- `ZOHOMAIL_TEST_DNS_BASE_DOMAIN`

Domain and DNS acceptance:

- `ZOHOMAIL_TEST_DNS_PROVIDER=cloudflare`
- `ZOHOMAIL_TEST_DNS_ZONE_NAME`
- `CLOUDFLARE_API_TOKEN`

Optional DNS tuning and overrides:

- `ZOHOMAIL_TEST_DNS_RESOLVER`
- `ZOHOMAIL_TEST_DNS_TIMEOUT`
- `ZOHOMAIL_TEST_ENABLE_SLOW_DNS_VERIFICATION`
- `ZOHOMAIL_TEST_ENABLE_ADVANCED_DOMAIN_FEATURES`
- `ZOHOMAIL_TEST_ENABLE_MAILBOX_LIFECYCLE`
- `ZOHOMAIL_TEST_ENABLE_MULTI_MAILBOX`
- `ZOHOMAIL_TEST_DNS_SPF_VALUE`
- `ZOHOMAIL_TEST_DNS_MX_10`
- `ZOHOMAIL_TEST_DNS_MX_20`
- `ZOHOMAIL_TEST_DNS_MX_50`

The acceptance tenant must be dedicated to tests and safe to mutate.

Zoho's official admin docs state that MX changes may take **6 to 24 hours** to take effect, and SPF or DKIM propagation may take **4 to 48 hours**. The default acceptance path therefore treats MX, SPF, and DKIM verification as opt-in slow checks. Set `ZOHOMAIL_TEST_ENABLE_SLOW_DNS_VERIFICATION=1` and raise `ZOHOMAIL_TEST_DNS_TIMEOUT` only when you intentionally want to run those long verification scenarios.

Some acceptance scenarios depend on tenant capabilities that are not universal across Zoho Mail plans. Set `ZOHOMAIL_TEST_ENABLE_ADVANCED_DOMAIN_FEATURES=1` to include the live `domain_catch_all` and `domain_subdomain_stripping` scenarios, set `ZOHOMAIL_TEST_ENABLE_MAILBOX_LIFECYCLE=1` only when the tenant has at least one spare mailbox license for mailbox create/import/update flows, and set `ZOHOMAIL_TEST_ENABLE_MULTI_MAILBOX=1` only when the tenant has enough mailbox licenses for multi-user flows such as mailbox forwarding and catch-all updates.

## Bootstrap The Local Acceptance Env

A local template is available in the repo root:

- `.env.testacc`

Load it into the current shell with:

```bash
set -a
source ./.env.testacc
set +a
```

To bootstrap or refresh Zoho OAuth tokens directly into that file, run:

```bash
make zoho-token
```

### Required Values And How To Retrieve Them

- `ZOHOMAIL_DATA_CENTER`
  Determine it from the Zoho Accounts or Zoho Mail URL you use as admin. Examples: `accounts.zoho.com` -> `us`, `accounts.zoho.eu` -> `eu`, `accounts.zoho.in` -> `in`. Keep the short provider code in the env file, not the full URL.
- `ZOHOMAIL_ORGANIZATION_ID`
  Open Zoho Mail Admin Console, go to `Organization` -> `Profile`, then read `Organization Info`. Zoho documents that this screen displays the organization ID.
- `ZOHOMAIL_CLIENT_ID`
  In the Zoho API Console, create a `Self Client` and copy the `Client ID` shown in the client secret tab.
- `ZOHOMAIL_CLIENT_SECRET`
  In the same Zoho API Console self client, copy the `Client Secret`.
- `ZOHOMAIL_AUTHORIZATION_CODE`
  Optional one-shot bootstrap input. In the Zoho API Console self client, use `Generate Code` with scopes `ZohoMail.organization.accounts.ALL,ZohoMail.organization.domains.ALL`, then paste the resulting authorization code here before running `make zoho-token`.
- `ZOHOMAIL_REFRESH_TOKEN`
  Durable token used to mint new access tokens. If it is already present, `make zoho-token` will reuse it and only refresh `ZOHOMAIL_ACCESS_TOKEN`.
- `ZOHOMAIL_ACCESS_TOKEN`
  Short-lived runtime token. `make zoho-token` updates it automatically from either `ZOHOMAIL_AUTHORIZATION_CODE` or `ZOHOMAIL_REFRESH_TOKEN`.
- `ZOHOMAIL_TEST_DNS_ZONE_NAME`
  This is the Cloudflare zone name you control for acceptance, for example `example.com` or `ansyo.ai`.
- `ZOHOMAIL_TEST_DNS_BASE_DOMAIN`
  Pick a delegated namespace under that zone that is safe to mutate, for example `tfacc.example.com`. The acceptance tests create random subdomains under this base domain.
- `CLOUDFLARE_API_TOKEN`
  In Cloudflare, create an API token scoped to the acceptance zone with DNS edit capability. A minimal token for these tests needs zone-scoped DNS read and edit permissions.

You do not need to provide the Zoho TXT verification hash in the env file. The provider exposes it directly as `zohomail_domain.txt_verification_value`, and the DNS-backed acceptance scenarios consume that computed value.

### Zoho OAuth Flow To Bootstrap The Tokens

1. Open the Zoho API Console and create a `Self Client`.
2. In `Generate Code`, request scopes `ZohoMail.organization.accounts.ALL,ZohoMail.organization.domains.ALL`.
3. Paste the generated code into `ZOHOMAIL_AUTHORIZATION_CODE` in `.env.testacc`.
4. Run `make zoho-token`.
5. The script exchanges that code against the right regional Zoho Accounts endpoint, writes `ZOHOMAIL_REFRESH_TOKEN` and `ZOHOMAIL_ACCESS_TOKEN`, then clears `ZOHOMAIL_AUTHORIZATION_CODE`.
6. On later runs, keep `ZOHOMAIL_REFRESH_TOKEN` and run `make zoho-token` again whenever you need a new `ZOHOMAIL_ACCESS_TOKEN`.

Example refresh request once you already have `client_id`, `client_secret`, and `refresh_token`:

```bash
curl -X POST "https://accounts.zoho.eu/oauth/v2/token" \
  -d "client_id=${ZOHOMAIL_CLIENT_ID}" \
  -d "client_secret=${ZOHOMAIL_CLIENT_SECRET}" \
  -d "grant_type=refresh_token" \
  -d "refresh_token=${ZOHOMAIL_REFRESH_TOKEN}"
```

## Minimum Acceptance Matrix

- `zohomail_mailbox`: create, import, `display_name` update, `role` update, replacement on create-only fields
- `zohomail_mailbox_alias`: create, import, delete, remote drift on alias removal
- `zohomail_mailbox_forwarding`: create, target update, `delete_zoho_mail_copy` update, import, delete, rejection of external domains
- `zohomail_domain`: create, import, delete, verification and hosting refresh
- `zohomail_domain_onboarding`: fast live path covers TXT verification, mail hosting, import, state-only delete
- `zohomail_domain_onboarding` slow opt-in: MX and SPF verification after extended DNS propagation
- `zohomail_domain_alias`: create, import, delete
- `zohomail_domain_dkim`: fast live path covers create, set default, import, delete
- `zohomail_domain_dkim` slow opt-in: public-key verification after extended DNS propagation
- `zohomail_domain_catch_all`: create, update, import, remote drift on removal
- `zohomail_domain_subdomain_stripping`: create, import, delete

## Examples And Docs

- `examples/resources/*/resource.tf` are the canonical usage examples.
- Any schema, example, or template change must be followed by `make generate`.
- Validate generated docs with `tfplugindocs validate`.

## Local Sonar

Sonar remains available for manual use:

- `make sonar-local`
- `make quality`
- `make quality-status`
- `make quality-reset`

It is advisory in this repo at this stage, not the default gate for pushes or PRs.

## Expected Evidence

When reporting validation, capture:

- the exact commands run
- whether the run was `unit/docs` only or `acceptance`
- which resource families were covered
- the result: `pass`, `skip by design`, or blocked by missing prerequisites
