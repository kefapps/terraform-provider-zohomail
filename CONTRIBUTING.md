# Contributing

Thanks for contributing to `terraform-provider-zohomail`.

This repository is intentionally scoped to a standalone Terraform provider for **Zoho Mail only**. Changes should keep the provider aligned with the public source address `kefapps/zohomail` and with normal Terraform Registry expectations.

## Before You Start

- Check whether an issue already exists for the bug, feature, or design change you want to work on.
- For non-trivial work, prefer agreeing the scope in an issue before opening a pull request.
- Keep changes focused. Avoid mixing provider behavior changes, release process changes, and broad repository housekeeping in the same pull request.

## Local Validation Gate

Run the default local gate before opening or updating a pull request:

```bash
make fmt
make test
make build
make generate
go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs validate --provider-name zohomail
```

This repository now mirrors the main pull request checks in GitHub Actions, but local validation still matters because acceptance coverage remains intentionally local-only.

## Acceptance Tests

Run `make testacc` when a change affects Zoho Mail API behavior, Terraform lifecycle behavior, import handling, idempotence, or drift detection.

Acceptance tests are local-only and require a dedicated Zoho Mail test tenant plus DNS credentials for the domain flows. Start from `.env.testacc.example`, copy it to `.env.testacc`, load it into your shell, and use `make zoho-token` if you need to refresh OAuth tokens.

Full acceptance guidance lives in [docs/ops/testing.md](docs/ops/testing.md).

## License And Contributions

This repository is published under `MPL-2.0`. By contributing to this repository, you agree that your contributions are submitted under the same license.

Keep existing SPDX license headers in source files, and add them to new provider source files when they are part of the distributed codebase.

## Documentation Expectations

- Keep `CHANGELOG.md` updated under `Unreleased` when the change is user-visible.
- Keep generated provider docs in `docs/` aligned with the current schema.
- Keep canonical examples in `examples/resources/*/resource.tf` aligned with real behavior.

If you change schema, examples, or documentation templates, rerun `make generate`.

## Pull Requests

Pull requests should include:

- a clear explanation of the behavior change
- the validation commands you ran
- any acceptance coverage that was run, skipped by design, or blocked by missing prerequisites
- documentation or example updates when user-facing behavior changed

Use the pull request template to keep the review surface explicit.
