package cmd

import (
	"context"
	"time"

	"github.com/spf13/cobra"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

func Execute() {
	if err := newCmd().Execute(); err != nil {
		log.Fatal(err)
	}
}
func newCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "operator",
		Short: "Manage operators in a cluster from the command line",
		Long: `Manage operators in a cluster from the command line.

kubectl operator helps you manage operator installations in your
cluster. It can install and uninstall operator catalogs, list
operators available for installation, and install and uninstall
operators from the installed catalogs.`,
	}

	var (
		cfg     action.Configuration
		timeout time.Duration
		cancel  context.CancelFunc
	)

	flags := cmd.PersistentFlags()
	cfg.BindFlags(flags)
	flags.DurationVar(&timeout, "timeout", 1*time.Minute, "The amount of time to wait before giving up on an operation.")

	cmd.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		var ctx context.Context
		ctx, cancel = context.WithTimeout(cmd.Context(), timeout)

		cmd.SetContext(ctx)

		return cfg.Load()
	}
	cmd.PersistentPostRun = func(command *cobra.Command, _ []string) {
		cancel()
	}

	cmd.AddCommand(
		newCatalogCmd(&cfg),
		newExtensionInstallCmd(&cfg),
		newExtensionUninstallCmd(&cfg),
		newExtensionListCmd(&cfg),
		//newExtensionListOperandsCmd(&cfg),
		newVersionCmd(),
	)

	return cmd
}
