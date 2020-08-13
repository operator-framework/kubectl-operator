.PHONY: all build gen-demo lint
all: build

build:
	go build -o bin/kubectl-operator

install: build
	install bin/kubectl-operator $(shell go env GOPATH)/bin

gen-demo:
	./assets/demo/gen-demo.sh

GOLANGCI_LINT_VER = "1.29.0"
lint:
#	scripts/golangci-lint-check.sh
ifneq (${GOLANGCI_LINT_VER}, "$(shell ./bin/golangci-lint --version 2>/dev/null | cut -b 27-32)")
	@echo "golangci-lint missing or not version '${GOLANGCI_LINT_VER}', downloading..."
	curl -sSfL "https://raw.githubusercontent.com/golangci/golangci-lint/v${GOLANGCI_LINT_VER}/install.sh" | sh -s -- -b ./bin "v${GOLANGCI_LINT_VER}"
endif
	./bin/golangci-lint --timeout 3m run
