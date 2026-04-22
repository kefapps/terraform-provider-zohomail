.PHONY: fmt test testacc generate install build coverage zoho-token release-check release-snapshot

GORELEASER ?= ./scripts/run-goreleaser.sh

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
