## Summary

Describe the change and why it is needed.

## Validation

- [ ] `make fmt`
- [ ] `make test`
- [ ] `make build`
- [ ] `make generate`
- [ ] `go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs validate --provider-name zohomail`
- [ ] `make testacc` when required by lifecycle, import, drift, or API behavior changes

## Checklist

- [ ] `CHANGELOG.md` updated under `Unreleased` when user-visible behavior changed
- [ ] docs and examples updated when schema or behavior changed
- [ ] acceptance scope explained if it was skipped or blocked
