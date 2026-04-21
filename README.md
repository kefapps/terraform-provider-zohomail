# terraform-provider-zohomail

`terraform-provider-zohomail` is a standalone Terraform provider bootstrap for **Zoho Mail only**.

This repository intentionally starts with a minimal HashiCorp Plugin Framework scaffold:

- provider source address: `kefapps/zohomail`
- provider local name: `zohomail`
- no provider authentication arguments yet
- no Zoho Mail resources or data sources yet

The goal of this bootstrap is to lock the project structure, local workflows, documentation generation, and Terraform Registry compatibility before implementing the Zoho Mail API surface.

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

The acceptance entrypoint already exists:

```bash
make testacc
```

At bootstrap stage, `make testacc` runs a provider smoke test only. Real Zoho Mail acceptance scenarios will be added once provider configuration and resources exist.

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

provider "zohomail" {}
```

For an unpublished provider, use `terraform plan` or `terraform apply` directly once the `dev_overrides` entry is in place. Do not rely on `terraform init` to install `kefapps/zohomail` from the public Registry before the provider is published there, because Terraform will still try to resolve the source address remotely.

## Documentation

Provider documentation is generated with `tfplugindocs`:

```bash
make generate
```

The provider index template lives in `templates/index.md.tmpl` and the generated Registry markdown is written to `docs/`.
