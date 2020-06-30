package cmd

import (
	"github.com/spf13/cobra"

	"github.com/joelanford/kubectl-operator/internal/pkg/action"
	"github.com/joelanford/kubectl-operator/internal/pkg/log"
)

func newCatalogUninstallCmd(cfg *action.Configuration) *cobra.Command {
	u := action.NewUninstallCatalog(cfg)
	cmd := &cobra.Command{
		Use:   "uninstall <catalog_name>",
		Short: "Uninstall an operator catalog",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			u.CatalogName = args[0]

			if err := u.Run(cmd.Context()); err != nil {
				log.Fatalf("failed to uninstall catalog %q: %v", u.CatalogName, err)
			}
			log.Printf("catalogsource %q uninstalled", u.CatalogName)
		},
	}

	return cmd
}
