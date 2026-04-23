# Changelog

All notable changes to this project will be documented in this file.

## Unreleased

- Add v1 Zoho Mail provider configuration and admin resources
- Document local testing and draft release policy for the provider
- Add OSS community files, ownership metadata, and GitHub contribution templates
- Add live acceptance coverage and verification hardening for mailbox and domain resources
- Split slow MX, SPF, and DKIM live verification out of the fast acceptance path and document the opt-in flow
- Add GoReleaser and GitHub Actions release scaffolding for the first public `kefapps/zohomail` release
- Document Terraform Registry, GPG, and local snapshot-release prerequisites
- Re-license the repository to Apache-2.0 and align OSS contribution, issue, and security metadata with the public GitHub setup
- Load release-signing secrets from 1Password in GitHub Actions and pin the local GoReleaser wrapper used by the release rehearsal targets
- Preserve mailbox role state when Zoho omits `roleName`, align mailbox acceptance language codes with Zoho's canonical lowercase form, and skip mailbox-heavy live scenarios cleanly on capacity-constrained tenants
