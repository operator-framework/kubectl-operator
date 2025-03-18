package cmd

import (
	"github.com/spf13/cobra"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/olmv1"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

func newOlmV1Cmd(cfg *action.Configuration) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "olmv1",
		Short: "Manage extensions via OLMv1 in a cluster from the command line",
		Long:  "Manage extensions via OLMv1 in a cluster from the command line.",
	}

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Display one or many resource(s)",
		Long:  "Display one or many resource(s)",
	}
	getCmd.AddCommand(
		olmv1.NewExtensionInstalledGetCmd(cfg),
		olmv1.NewCatalogInstalledGetCmd(cfg),
	)

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a resource",
		Long:  "Create a resource",
	}
	createCmd.AddCommand(olmv1.NewCatalogCreateCmd(cfg))

	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a resource",
		Long:  "Delete a resource",
	}
	deleteCmd.AddCommand(olmv1.NewCatalogDeleteCmd(cfg))

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update a resource",
		Long:  "Update a resource",
	}
	updateCmd.AddCommand(
		olmv1.NewExtensionUpdateCmd(cfg),
	)

	cmd.AddCommand(
		olmv1.NewExtensionInstallCmd(cfg),
		olmv1.NewExtensionUninstallCmd(cfg),
		getCmd,
		createCmd,
		deleteCmd,
		updateCmd,
	)

	return cmd
}
