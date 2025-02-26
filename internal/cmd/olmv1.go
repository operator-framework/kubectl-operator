package cmd

import (
	"github.com/spf13/cobra"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/olmv1"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

func newOlmV1Cmd(cfg *action.Configuration) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "olmv1",
		Short: "Manage operators via OLMv1 in a cluster from the command line",
		Long:  "Manage operators via OLMv1 in a cluster from the command line.",
	}

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Display one or many OLMv1-specific resource(s)",
		Long:  "Display one or many OLMv1-specific resource(s)",
	}
	getCmd.AddCommand(
		olmv1.NewOperatorInstalledGetCmd(cfg),
		olmv1.NewCatalogInstalledGetCmd(cfg),
	)

	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete an OLMv1 resource",
		Long:  "Delete an OLMv1 resource",
	}
	deleteCmd.AddCommand(olmv1.NewCatalogDeleteCmd(cfg))

	cmd.AddCommand(
		olmv1.NewOperatorInstallCmd(cfg),
		olmv1.NewOperatorUninstallCmd(cfg),
		getCmd,
		deleteCmd,
	)

	return cmd
}
