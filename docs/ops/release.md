# Release Policy

`terraform-provider-zohomail` now includes the baseline release automation needed for a public Terraform provider release, but actual publication is still gated on external prerequisites.

## Repo Release Assets

The repo now includes:

- `.github/workflows/release.yml` to create a GitHub Release on `v*` tags
- `.goreleaser.yml` to build Terraform provider archives, checksums, and signed checksum files
- `terraform-registry-manifest.json` bundled into release assets for Terraform Registry consumers

## External Prerequisites Before The First Public Release

Before cutting a real public tag, make sure all of the following are in place:

- the GitHub repository is public under the `kefapps` namespace
- GitHub Actions secrets `GPG_PRIVATE_KEY` and `PASSPHRASE` are configured
- the matching GPG public key is uploaded to Terraform Registry
- the provider `kefapps/zohomail` has been onboarded in Terraform Registry

Do not assume a published provider exists until those four prerequisites are satisfied.

## What Must Stay True Before Any Public Release

- `CHANGELOG.md` is updated under `Unreleased`
- generated docs in `docs/` match the current schema
- `examples/resources/*/resource.tf` remain accurate
- the local validation gate passes:

```bash
make test
make build
make generate
go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs validate --provider-name zohomail
```

- the relevant live acceptance scenarios have been run locally when lifecycle, import, drift, or API behavior changed

## Local Release Rehearsal

If `goreleaser` is installed on your machine, use these commands:

```bash
make release-check
make release-snapshot
```

`make release-snapshot` intentionally skips `publish` and `sign` so the release packaging can be rehearsed locally without GitHub credentials or a local GPG signing setup.

## Versioning Contract

Public releases follow this contract:

- provider source address: `kefapps/zohomail`
- semantic version tags prefixed with `v`
- tags cut from `main`
- first public release target: `v0.1.0`

The provider should remain in `0.x` until the public release process is validated end to end.

## Tagging And Publish Flow

Once the external prerequisites are in place and `main` is validated:

1. update `CHANGELOG.md`
2. create and push a tag such as `v0.1.0`
3. let `.github/workflows/release.yml` build the artifacts and create the GitHub Release
4. complete the Terraform Registry-side publication flow if this is the first release
