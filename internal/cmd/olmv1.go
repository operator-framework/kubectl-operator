package cmd

import (
	"github.com/spf13/cobra"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/olmv1/catalog"
	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/olmv1/operator"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

func newOlmV1Cmd(cfg *action.Configuration) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "olmv1",
		Short: "Manage operators via OLMv1 in a cluster from the command line",
		Long:  "Manage operators via OLMv1 in a cluster from the command line.",
	}

	cmd.AddCommand(
		operator.NewOperatorInstallCmd(cfg),
		operator.NewOperatorUninstallCmd(cfg),
		catalog.NewCatalogCommand(cfg),
	)

	return cmd
}
