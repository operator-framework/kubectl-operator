SHELL:=/bin/bash

export GIT_VERSION = $(shell git describe --tags --always)
export GIT_COMMIT = $(shell git rev-parse HEAD)
export GIT_COMMIT_TIME = $(shell TZ=UTC git show -s --format=%cd --date=format-local:%Y-%m-%dT%TZ)
export GIT_TREE_STATE = $(shell sh -c '(test -n "$(shell git status -s)" && echo "dirty") || echo "clean"')
export CGO_ENABLED = 1

REPO = $(shell go list -m)
GO_BUILD_ARGS = \
  -gcflags "all=-trimpath=$(shell dirname $(shell pwd))" \
  -asmflags "all=-trimpath=$(shell dirname $(shell pwd))" \
  -ldflags " \
    -s \
    -w \
    -X '$(REPO)/internal/version.GitVersion=$(GIT_VERSION)' \
    -X '$(REPO)/internal/version.GitCommit=$(GIT_COMMIT)' \
    -X '$(REPO)/internal/version.GitCommitTime=$(GIT_COMMIT_TIME)' \
    -X '$(REPO)/internal/version.GitTreeState=$(GIT_TREE_STATE)' \
  " \

.PHONY: all
all: install

.PHONY: vet
vet:
	go vet ./...

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: build
build: vet fmt
	go build $(GO_BUILD_ARGS) -o bin/kubectl-operator

.PHONY: test
test:
	go test ./...

.PHONY: install
install: build
	install bin/kubectl-operator $(shell go env GOPATH)/bin

.PHONY: gen-demo
gen-demo:
	./assets/demo/gen-demo.sh

.PHONY: lint
lint:
	source ./scripts/fetch.sh; fetch golangci-lint 1.50.1 && ./bin/golangci-lint --timeout 3m run

.PHONY: release
RELEASE_ARGS?=release --rm-dist --snapshot
release:
	source ./scripts/fetch.sh; fetch goreleaser 1.13.0 && ./bin/goreleaser $(RELEASE_ARGS)
