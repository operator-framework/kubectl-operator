package cmd

import (
	"github.com/spf13/cobra"

	"github.com/joelanford/kubectl-operator/internal/pkg/action"
)

func newCatalogCmd(cfg *action.Configuration) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "catalog",
		Short: "Manage operator catalogs",
	}
	cmd.AddCommand(
		newCatalogInstallCmd(cfg),
		newCatalogListCmd(cfg),
		newCatalogUninstallCmd(cfg),
	)
	return cmd
}
