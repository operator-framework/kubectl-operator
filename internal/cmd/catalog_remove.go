package cmd

import (
	"github.com/spf13/cobra"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	"github.com/operator-framework/kubectl-operator/internal/pkg/action"
)

func newCatalogRemoveCmd(cfg *action.Configuration) *cobra.Command {
	u := action.NewCatalogRemove(cfg)
	cmd := &cobra.Command{
		Use:   "remove <catalog_name>",
		Short: "Remove a operator catalog",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			u.CatalogName = args[0]

			if err := u.Run(cmd.Context()); err != nil {
				log.Fatalf("failed to remove catalog %q: %v", u.CatalogName, err)
			}
			log.Printf("catalogsource %q removed", u.CatalogName)
		},
	}

	return cmd
}
