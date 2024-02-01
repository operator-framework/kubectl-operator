package catalog

import (
	"github.com/operator-framework/kubectl-operator/pkg/action"
	"github.com/spf13/cobra"
)

func NewCatalogCommand(cfg *action.Configuration) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "catalog",
		Short: "list, inspect, and search for content in a catalog",
		Long:  "CLI for listing, inspecting, and searching for content provided by catalogd's Catalog resources",
	}

	cmd.AddCommand(NewListCommand(cfg))
	cmd.AddCommand(NewSearchCommand(cfg))
	cmd.AddCommand(NewInspectCommand(cfg))
	return cmd
}
