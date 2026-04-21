.PHONY: fmt test testacc generate install build coverage sonar-local quality quality-status quality-reset

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

sonar-local: coverage
	./scripts/sast-sonarqube.sh

quality:
	./scripts/quality-pr-gate.sh

quality-status:
	node ./scripts/quality-sonarqube-local.mjs status

quality-reset:
	node ./scripts/quality-sonarqube-local.mjs reset
