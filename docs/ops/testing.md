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
- `ZOHOMAIL_TEST_DNS_VERIFICATION_TARGET`
- `ZOHOMAIL_TEST_DNS_SPF_VALUE`
- `ZOHOMAIL_TEST_DNS_MX_10`
- `ZOHOMAIL_TEST_DNS_MX_20`
- `ZOHOMAIL_TEST_DNS_MX_50`

The acceptance tenant must be dedicated to tests and safe to mutate.

## Minimum Acceptance Matrix

- `zohomail_mailbox`: create, import, `display_name` update, `role` update, replacement on create-only fields
- `zohomail_mailbox_alias`: create, import, delete, remote drift on alias removal
- `zohomail_mailbox_forwarding`: create, target update, `delete_zoho_mail_copy` update, import, delete, rejection of external domains
- `zohomail_domain`: create, import, delete, verification and hosting refresh
- `zohomail_domain_onboarding`: verification, hosting/SPF/MX/primary toggles, import, state-only delete
- `zohomail_domain_alias`: create, import, delete
- `zohomail_domain_dkim`: create, set default, verify, import, delete
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
