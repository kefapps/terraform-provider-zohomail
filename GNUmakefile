.PHONY: fmt test testacc generate install build coverage sonar-local quality quality-status quality-reset zoho-token release-check release-snapshot

GORELEASER ?= goreleaser

fmt:
	go fmt ./...
	terraform fmt -recursive ./examples/

test:
	go test ./...

coverage:
	go test -coverprofile=coverage.out ./...

testacc:
	TF_ACC=1 go test -v ./...

generate:
	go generate ./...

install:
	go install .

build:
	go build ./...

release-check:
	$(GORELEASER) check

release-snapshot:
	$(GORELEASER) release --snapshot --clean --skip=publish,sign

zoho-token:
	./scripts/zoho-oauth-token.sh --env-file ./.env.testacc

sonar-local: coverage
	./scripts/sast-sonarqube.sh

quality:
	./scripts/quality-pr-gate.sh

quality-status:
	node ./scripts/quality-sonarqube-local.mjs status

quality-reset:
	node ./scripts/quality-sonarqube-local.mjs reset
