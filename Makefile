.PHONY: all build
all: build

build:
	go build -o bin/kubectl-operator

install: build
	install bin/kubectl-operator $(shell go env GOPATH)/bin
