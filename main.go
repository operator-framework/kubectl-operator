package main

import (
	"github.com/operator-framework/kubectl-operator/internal/cmd"
	"github.com/operator-framework/kubectl-operator/internal/pkg/log"
)

func main() {
	if err := cmd.New().Execute(); err != nil {
		log.Fatal(err)
	}
}
