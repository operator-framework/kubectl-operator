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
	deleteCmd.AddCommand(
		olmv1.NewCatalogDeleteCmd(cfg),
		olmv1.NewExtensionDeleteCmd(cfg),
	)

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update a resource",
		Long:  "Update a resource",
	}
	updateCmd.AddCommand(
		olmv1.NewExtensionUpdateCmd(cfg),
		olmv1.NewCatalogUpdateCmd(cfg),
	)

	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Install a resource",
		Long:  "Install a resource",
	}
	installCmd.AddCommand(olmv1.NewExtensionInstallCmd(cfg))

	searchCmd := &cobra.Command{
		Use:   "search",
		Short: "Search for packages",
		Long:  "Search one or all available catalogs for packages or versions",
	}
	searchCmd.AddCommand(olmv1.NewCatalogSearchCmd(cfg))

	cmd.AddCommand(
		installCmd,
		getCmd,
		createCmd,
		deleteCmd,
		updateCmd,
		searchCmd,
	)

	return cmd
}
