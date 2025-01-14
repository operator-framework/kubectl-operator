.DEFAULT_GOAL := build

SHELL:=/bin/bash

export GIT_VERSION = $(shell git describe --tags --always)
export GIT_COMMIT = $(shell git rev-parse HEAD)
export GIT_COMMIT_TIME = $(shell TZ=UTC git show -s --format=%cd --date=format-local:%Y-%m-%dT%TZ)
export GIT_TREE_STATE = $(shell sh -c '(test -n "$(shell git status -s)" && echo "dirty") || echo "clean"')
export CGO_ENABLED = 1

# bingo manages consistent tooling versions for things like kind, kustomize, etc.
include .bingo/Variables.mk

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

.PHONY: vet
vet:
	go vet ./...

.PHONY: fmt
fmt:
	go fmt ./...


.PHONY: bingo-upgrade
bingo-upgrade: $(BINGO) #EXHELP Upgrade tools
	@for pkg in $$($(BINGO) list | awk '{ print $$3 }' | tail -n +3 | sed 's/@.*//'); do \
		echo -e "Upgrading \033[35m$$pkg\033[0m to latest..."; \
		$(BINGO) get "$$pkg@latest"; \
	done

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
lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) --timeout 3m run

.PHONY: release
RELEASE_ARGS?=release --clean --snapshot
release: $(GORELEASER)
	$(GORELEASER) $(RELEASE_ARGS)
