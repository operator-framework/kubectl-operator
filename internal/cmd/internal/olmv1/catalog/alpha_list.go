package catalog

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/operator-framework/kubectl-operator/internal/pkg/catalogd/fetcher"
	"github.com/operator-framework/kubectl-operator/internal/pkg/catalogd/streamer"
	experimentalaction "github.com/operator-framework/kubectl-operator/internal/pkg/experimental/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"

	"github.com/spf13/cobra"
)

func NewListCommand(cfg *action.Configuration) *cobra.Command {
	i := experimentalaction.NewCatalogList(cfg,
		func(c *action.Configuration) experimentalaction.CatalogFetcher {
			return fetcher.New(c.Client)
		},
		func(c *action.Configuration) experimentalaction.CatalogContentStreamer {
			return streamer.New(c.Clientset.CoreV1())
		},
	)
	i.Logf = log.Printf

	listOpts := experimentalaction.CatalogListOptions{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "list catalog objects",
		RunE: func(cmd *cobra.Command, args []string) error {
			return list(cmd.Context(), i, listOpts)
		},
	}

	cmd.Flags().StringVar(&listOpts.Catalog, "catalog", "", "filter results to only be from the specified catalog")
	cmd.Flags().StringVar(&listOpts.Schema, "schema", "", "filter results to only be FBC objects that have the specified schema")
	cmd.Flags().StringVar(&listOpts.Package, "package", "", "filter results to only be FBC objects that belong to the specified package")
	cmd.Flags().StringVar(&listOpts.Name, "name", "", "filter results to only be FBC objects with the specified name")

	return cmd
}

func list(ctx context.Context, lister *experimentalaction.CatalogList, listOpts experimentalaction.CatalogListOptions) error {
	if lister == nil {
		return errors.New("nil CatalogList action provided")
	}
	metas, err := lister.Run(ctx, listOpts)
	if err != nil {
		return fmt.Errorf("performing list: %w", err)
	}

	out := strings.Builder{}
	for _, meta := range metas {
		out.WriteString(CatalogNameStyle.Render(meta.Catalog) + " ")
		out.WriteString(SchemaNameStyle.Render(meta.Schema) + " ")
		out.WriteString(PackageNameStyle.Render(meta.Package) + " ")
		out.WriteString(NameStyle.Render(meta.Name))
		out.WriteString("\n")
	}
	fmt.Print(out.String())
	return nil
}
