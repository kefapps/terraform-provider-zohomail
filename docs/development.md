# Development

This repository follows a local-first development model for ordinary provider work and keeps live Zoho Mail acceptance tests local-only.

## Default Validation Gate

Run these commands before opening or updating a pull request:

```bash
make fmt
make test
make build
make generate
go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs validate --provider-name zohomail
```

GitHub Actions is expected to pass the `Build`, `Unit Tests`, `Generate Check`, and `Docs Validate` checks.

## Documentation Workflow

Provider documentation is generated with `tfplugindocs`:

```bash
make generate
```

The provider index template lives in `templates/index.md.tmpl`, and generated Registry markdown is written to `docs/`.

`examples/resources/*/resource.tf` are the canonical example snippets and should stay aligned with both schema behavior and generated docs.

## Acceptance Tests

Run acceptance tests when a change touches Zoho Mail API behavior, Terraform lifecycle behavior, import handling, idempotence, or drift detection:

```bash
make testacc
```

Acceptance tests require a dedicated Zoho Mail tenant and, for domain flows, DNS credentials. The full setup and scenario matrix live in [ops/testing.md](ops/testing.md).

To bootstrap or refresh Zoho OAuth tokens in a local `.env.testacc` created from `.env.testacc.example`, run:

```bash
make zoho-token
```

## Contributing And Security

- License and contribution terms: [../LICENSE](../LICENSE)
- Contribution workflow and validation expectations: [../CONTRIBUTING.md](../CONTRIBUTING.md)
- Vulnerability reporting posture: [../SECURITY.md](../SECURITY.md)
- Maintainer and automation guardrails: [../AGENTS.md](../AGENTS.md)

Use the GitHub issue templates for bugs and feature requests. Do not open a public issue for a suspected security vulnerability.
