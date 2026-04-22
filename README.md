# terraform-provider-zohomail

`terraform-provider-zohomail` is a standalone Terraform provider for **Zoho Mail only**.

This repository targets the public provider source address `kefapps/zohomail` and implements an admin-focused v1 surface on top of the official Zoho Mail APIs.

## Requirements

- Go `>= 1.25.8`
- Terraform `>= 1.14`

## Build and Test

```bash
make fmt
make test
make build
make generate
go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs validate
```

This is the default local validation gate for normal provider work. There is no GitHub Actions CI gate yet.

## Acceptance Tests

The acceptance entrypoint exists for live Zoho Mail environments:

```bash
make testacc
```

Acceptance runs are **local only** for now. They are expected when a change touches Zoho Mail API behavior, Terraform lifecycle, import handling, idempotence, or drift detection.

To bootstrap or refresh Zoho OAuth tokens in `.env.testacc`, run:

```bash
make zoho-token
```

Base acceptance environment:

- `TF_ACC=1`
- `ZOHOMAIL_ACCESS_TOKEN`
- `ZOHOMAIL_DATA_CENTER`
- `ZOHOMAIL_ORGANIZATION_ID`

Any scenario that creates disposable Zoho domains or mailboxes also needs:

- `ZOHOMAIL_TEST_DNS_BASE_DOMAIN`

Domain and DNS acceptance environment:

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

Zoho's official admin docs state that MX changes may take **6 to 24 hours** to take effect, and SPF or DKIM propagation may take **4 to 48 hours**. The default live suite therefore keeps MX, SPF, and DKIM verification out of the fast path. Set `ZOHOMAIL_TEST_ENABLE_SLOW_DNS_VERIFICATION=1` and increase `ZOHOMAIL_TEST_DNS_TIMEOUT` if you explicitly want to run those long verification checks.

Some tenant capabilities are intentionally opt-in because they are not guaranteed on every Zoho Mail plan. Set `ZOHOMAIL_TEST_ENABLE_ADVANCED_DOMAIN_FEATURES=1` to run the live `domain_catch_all` and `domain_subdomain_stripping` scenarios, set `ZOHOMAIL_TEST_ENABLE_MAILBOX_LIFECYCLE=1` when the tenant has at least one spare mailbox license for create/import/update mailbox flows, and set `ZOHOMAIL_TEST_ENABLE_MULTI_MAILBOX=1` when the tenant has enough mailbox licenses for scenarios that create multiple users in the same run.

The acceptance suite currently implements these live scenarios:

- `zohomail_mailbox`: opt-in mailbox-lifecycle path covers create, import, update `display_name`, update `role`, and replacement on create-only fields
- `zohomail_mailbox_alias`: opt-in mailbox-lifecycle path covers create, import, delete, and drift when the alias disappears remotely
- `zohomail_mailbox_forwarding`: mailbox-lifecycle path covers rejection of external domains; the multi-mailbox path for create, update, import, and delete is opt-in via `ZOHOMAIL_TEST_ENABLE_MULTI_MAILBOX=1`
- `zohomail_domain`: create, import, delete, refresh of verification and hosting state
- `zohomail_domain_onboarding`: fast live path covers TXT verification, mail hosting, import, state-only delete; slow opt-in path covers MX and SPF verification after extended propagation
- `zohomail_domain_alias`: create, import, delete
- `zohomail_domain_dkim`: fast live path covers create, set default, import, delete; slow opt-in path covers public-key verification after extended propagation
- `zohomail_domain_catch_all`: opt-in advanced-feature and multi-mailbox path covers create, update, import, and drift when the catch-all disappears remotely
- `zohomail_domain_subdomain_stripping`: opt-in advanced-feature path covers create, import, and delete

## Provider Configuration

```terraform
terraform {
  required_providers {
    zohomail = {
      source = "kefapps/zohomail"
    }
  }
}

provider "zohomail" {
  organization_id = var.zohomail_organization_id
  access_token    = var.zohomail_access_token
  data_center     = var.zohomail_data_center
}
```

All three arguments also support environment fallbacks:

- `ZOHOMAIL_ORGANIZATION_ID`
- `ZOHOMAIL_ACCESS_TOKEN`
- `ZOHOMAIL_DATA_CENTER`

Supported `data_center` values are: `us`, `eu`, `in`, `au`, `jp`, `ca`, `cn`, `ae`, `sa`.

## V1 Resources

- `zohomail_mailbox`
- `zohomail_mailbox_alias`
- `zohomail_mailbox_forwarding`
- `zohomail_domain`
- `zohomail_domain_onboarding`
- `zohomail_domain_alias`
- `zohomail_domain_dkim`
- `zohomail_domain_catch_all`
- `zohomail_domain_subdomain_stripping`

The user-facing need “plusieurs adresses du même domaine arrivent sur un seul compte” is handled primarily via `zohomail_mailbox_alias`.

`zohomail_mailbox_forwarding` is intentionally narrower in v1:

- it manages forwarding targets for a mailbox
- it only accepts target addresses that belong to domains already attached to that mailbox
- it does not attempt external forwarding verification flows

## Install Locally

Install the provider binary into your Go bin directory:

```bash
make install
```

For local Terraform development, install the provider binary and point Terraform to the directory that contains it. If `GOBIN` is unset, this is usually `$(go env GOPATH)/bin`.

Use a CLI config file with a development override:

```hcl
provider_installation {
  dev_overrides {
    "kefapps/zohomail" = "/path/to/your/go/bin"
  }

  direct {}
}
```

Then use the provider in Terraform:

```terraform
terraform {
  required_providers {
    zohomail = {
      source = "kefapps/zohomail"
    }
  }
}

provider "zohomail" {
  organization_id = var.zohomail_organization_id
  access_token    = var.zohomail_access_token
  data_center     = var.zohomail_data_center
}
```

For an unpublished provider, use `terraform plan` or `terraform apply` directly once the `dev_overrides` entry is in place. Do not rely on `terraform init` to install `kefapps/zohomail` from the public Registry before the provider is published there, because Terraform will still try to resolve the source address remotely.

## Local Sonar Quality Gate

This repository follows the same local quality-gate discipline as `../goose/keftionnaire`:

- `make sonar-local`: local Sonar scan for work-in-progress debugging
- `make quality`: strict local certification on a clean worktree, intended before push
- `make quality-status`: inspect the local SonarQube stack state
- `make quality-reset`: stop and clean the local SonarQube stack state

These commands are available for local diagnosis and stricter manual certification. They are not the default repo gate and they are not mirrored in GitHub CI at this stage.

`make quality`:

- refuses a dirty worktree
- generates Go coverage
- boots a local SonarQube stack through Docker Compose
- waits for the local quality gate result

## Documentation

Provider documentation is generated with `tfplugindocs`:

```bash
make generate
```

The provider index template lives in `templates/index.md.tmpl` and the generated Registry markdown is written to `docs/`.

`examples/resources/*/resource.tf` are the canonical example snippets and should stay aligned with both schema behavior and generated docs.

Operational runbooks:

- `docs/ops/testing.md`
- `docs/ops/release.md`

## Release Policy

Release automation is intentionally not in place yet:

- no `.github/workflows` CI or release pipeline
- no `.goreleaser.yml`
- no automated Terraform Registry publication

Current release posture:

- keep `CHANGELOG.md` updated under `Unreleased`
- keep docs/examples/generated markdown current on each provider change
- treat future provider tags as `v*` semver tags cut from `main`
- reserve the first public provider release for `v0.1.0` once the v1 acceptance matrix is implemented and documented
