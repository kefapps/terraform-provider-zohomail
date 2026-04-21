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
```

The acceptance entrypoint exists for live Zoho Mail environments:

```bash
make testacc
```

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

`make quality` is mandatory before push. It:

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
