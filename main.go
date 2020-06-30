package main

import (
	"github.com/joelanford/kubectl-operator/internal/cmd"
	"github.com/joelanford/kubectl-operator/internal/pkg/log"
)

func main() {
	if err := cmd.New().Execute(); err != nil {
		log.Fatal(err)
	}
}
