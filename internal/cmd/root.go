package cmd

import (
	"github.com/spf13/cobra"

	"github.com/joelanford/kubectl-operator/internal/pkg/action"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "operator",
		Short: "Manage operators in a cluster from the command line",
		Long: `Manage operators in a cluster from the command line.

kubectl operator helps you manage operator installations in your
cluster. It can install and uninstall operator catalogs, list
operators available for installation, and install and uninstall
operators from the installed catalogs.`,
	}

	flags := cmd.PersistentFlags()

	var cfg action.Configuration
	cfg.BindFlags(flags)

	cmd.PersistentPreRunE = func(*cobra.Command, []string) error {
		return cfg.Load()
	}

	cmd.AddCommand(
		newCatalogCmd(&cfg),
		newOperatorInstallCmd(&cfg),
		newOperatorUpgradeCmd(&cfg),
		newOperatorUninstallCmd(&cfg),
		newOperatorListCmd(&cfg),
		newOperatorListAvailableCmd(&cfg),
	)

	return cmd
}
