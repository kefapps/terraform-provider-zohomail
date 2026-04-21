# Release Policy

`terraform-provider-zohomail` is not ready for automated public release yet.

## Current State

The repo intentionally does **not** include:

- GitHub Actions CI or release workflows
- `.goreleaser.yml`
- automated Terraform Registry publication

Release work is still draft-level and local-first.

## What Must Stay True Before Any Public Release

- `CHANGELOG.md` is updated under `Unreleased`
- generated docs in `docs/` match the current schema
- `examples/resources/*/resource.tf` remain accurate
- the local validation gate passes:

```bash
make test
make build
make generate
go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs validate
```

- the relevant live acceptance scenarios have been run locally when lifecycle, import, drift, or API behavior changed

## Versioning Contract

When public releases are introduced later, they should follow:

- provider source address: `kefapps/zohomail`
- semantic version tags prefixed with `v`
- tags cut from `main`
- first public release target: `v0.1.0`

The provider should remain in `0.x` while the v1 acceptance matrix is still incomplete or not yet automated/documented.

## Scope Of A Future Release Ticket

The future implementation ticket can add:

- `.github/workflows` for build and release
- `.goreleaser.yml`
- signed provider artifacts for supported platforms
- GitHub Release creation
- Terraform Registry publication flow

None of that is part of the current repo contract.

## Operator Posture

For now, treat every release discussion as one of:

- draft planning for future publication
- local validation of a merge candidate
- manual preparation of changelog and versioning policy

Do not assume a published provider exists until the release workflow is explicitly implemented.
