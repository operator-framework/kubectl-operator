package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/olmv1"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

func Execute() {
	if err := newCmd().Execute(); err != nil {
		log.Fatal(err)
	}
}

const (
	experimentalOLMV1EnvVar = "EXPERIMENTAL_USE_OLMV1_APIS"
)

func newCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "operator",
		Short: "Manage operators in a cluster from the command line",
		Long: fmt.Sprintf(`Manage operators in a cluster from the command line.

kubectl operator helps you manage operator installations in your
cluster. It can install and uninstall operator catalogs, list
operators available for installation, and install and uninstall
operators from the installed catalogs.

To try out experimental OLMv1 APIs, set "%s=on".
`, experimentalOLMV1EnvVar),
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

	if v := os.Getenv(experimentalOLMV1EnvVar); v == "on" {
		cmd.AddCommand(
			olmv1.NewOperatorInstallCmd(&cfg),
			olmv1.NewOperatorUninstallCmd(&cfg),
		)
		return cmd
	}

	cmd.AddCommand(
		newCatalogCmd(&cfg),
		newOperatorInstallCmd(&cfg),
		newOperatorUpgradeCmd(&cfg),
		newOperatorUninstallCmd(&cfg),
		newOperatorListCmd(&cfg),
		newOperatorListAvailableCmd(&cfg),
		newOperatorListOperandsCmd(&cfg),
		newOperatorDescribeCmd(&cfg),
		newVersionCmd(),
	)

	return cmd
}
