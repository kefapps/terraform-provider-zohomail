.PHONY: fmt test testacc generate install build

fmt:
	go fmt ./...
	terraform fmt -recursive ./examples/

test:
	go test ./...

testacc:
	TF_ACC=1 go test -v ./...

generate:
	go generate ./...

install:
	go install .

build:
	go build ./...
