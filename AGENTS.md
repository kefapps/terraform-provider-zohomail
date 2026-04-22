# AGENTS.md

This file contains maintainer and automation guardrails for this repository.
Human contributors should start with `README.md` and `CONTRIBUTING.md`.

## Repo Guardrails

- Scope: stay within the standalone Zoho Mail Terraform provider. Do not widen the repo back to Ansyo or a generic Zoho provider.
- Docs: read `README.md` first, then the specific runbook or source file relevant to the task.
- Provider work: keep a conventional open-source posture compatible with Terraform Registry expectations.
- Testing and release policy: follow `docs/ops/testing.md` and `docs/ops/release.md` when work touches quality gates, acceptance tests, or publication.
- Docs generation: rerun `make generate` after any schema, example, or documentation template change.

Details: `README.md`
