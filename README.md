# Terraform Provider for Zoho Mail

> Disclaimer: this project is independent, unaffiliated with Zoho, and not endorsed by Zoho Corporation. I am building it for fun, usefulness, and the strange joy of making email administration declarative.
>
> Dear Zoho team: this provider is handcrafted for free; a lifetime Zoho One license would be a delightfully practical thank-you, and would make it much easier to teach Terraform how to manage the rest of the Zoho universe.

`terraform-provider-zohomail` is a standalone Terraform provider for **Zoho Mail administration**.

It targets the public provider source address `kefapps/zohomail` and focuses on the v1 admin workflows that are useful to infrastructure teams: domains, onboarding, mailboxes, aliases, internal forwarding, DKIM, catch-all, and subdomain stripping.

The repository is licensed under [Apache-2.0](LICENSE).

## Quick Links

- [Provider documentation](docs/index.md)
- [Getting started](docs/getting-started.md)
- [Development guide](docs/development.md)
- [Testing and acceptance policy](docs/ops/testing.md)
- [Release policy](docs/ops/release.md)
- [Contributing](CONTRIBUTING.md)
- [Security policy](SECURITY.md)

## Example Usage

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

The provider also reads these environment variables:

- `ZOHOMAIL_ORGANIZATION_ID`
- `ZOHOMAIL_ACCESS_TOKEN`
- `ZOHOMAIL_DATA_CENTER`

Supported `data_center` values are `us`, `eu`, `in`, `au`, `jp`, `ca`, `cn`, `ae`, and `sa`.

## Resources

The current v1 surface includes:

- `zohomail_mailbox`
- `zohomail_mailbox_alias`
- `zohomail_mailbox_forwarding`
- `zohomail_domain`
- `zohomail_domain_onboarding`
- `zohomail_domain_alias`
- `zohomail_domain_dkim`
- `zohomail_domain_catch_all`
- `zohomail_domain_subdomain_stripping`

`zohomail_mailbox_alias` is the primary answer for the common need: “several addresses on the same domain should land in one mailbox.”

## Local Development

Requirements:

- Go `>= 1.25.8`
- Terraform `>= 1.14`

Run the normal local validation gate with:

```bash
make fmt
make test
make build
make generate
go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs validate --provider-name zohomail
```

Use `make install` when you want to install the provider binary locally. See [Getting started](docs/getting-started.md) for local Terraform usage and development overrides.

## Documentation

Generated Terraform Registry markdown lives under `docs/`, with canonical snippets in `examples/resources/*/resource.tf`.

Run `make generate` after any schema, example, or documentation template change. The provider index template lives in [templates/index.md.tmpl](templates/index.md.tmpl).

## Release Status

Release automation is present, but public publication is still gated on Terraform Registry onboarding and release-signing setup. The first public release target is `v0.1.0`.

See [docs/ops/release.md](docs/ops/release.md) for the full release checklist.
