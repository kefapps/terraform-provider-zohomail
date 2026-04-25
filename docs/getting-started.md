# Getting Started

This guide covers the short path from a local provider build to a Terraform configuration that can talk to Zoho Mail.

## Requirements

- Go `>= 1.25.8`
- Terraform `>= 1.14`
- a Zoho Mail organization where you can create an OAuth token with admin scopes

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

All three provider arguments can also be supplied through environment variables:

- `ZOHOMAIL_ORGANIZATION_ID`
- `ZOHOMAIL_ACCESS_TOKEN`
- `ZOHOMAIL_DATA_CENTER`

Supported `data_center` values are:

- `us`
- `eu`
- `in`
- `au`
- `jp`
- `ca`
- `cn`
- `ae`
- `sa`

## Local Install

Install the provider binary into your Go bin directory:

```bash
make install
```

For local Terraform development, install the provider binary and point Terraform to the directory that contains it. If `GOBIN` is unset, this is usually `$(go env GOPATH)/bin`.

Use a Terraform CLI config file with a development override:

```hcl
provider_installation {
  dev_overrides {
    "kefapps/zohomail" = "/path/to/your/go/bin"
  }

  direct {}
}
```

Then run `terraform plan` or `terraform apply` directly once the `dev_overrides` entry is in place.

Do not rely on `terraform init` to install `kefapps/zohomail` from the public Registry before the provider is published there. Terraform will still try to resolve the source address remotely.

## Current V1 Scope

The provider is intentionally scoped to Zoho Mail administration:

- mailboxes
- mailbox aliases
- internal mailbox forwarding
- domains
- domain onboarding
- domain aliases
- DKIM
- catch-all
- subdomain stripping

`zohomail_mailbox_forwarding` is intentionally narrower in v1:

- it manages forwarding targets for a mailbox
- it only accepts target addresses that belong to domains already attached to that mailbox
- it does not attempt external forwarding verification flows

For live acceptance setup, OAuth token refresh, DNS-backed domain scenarios, and the minimum acceptance matrix, see [ops/testing.md](ops/testing.md).
