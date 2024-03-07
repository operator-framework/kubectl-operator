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

func NewSearchCommand(cfg *action.Configuration) *cobra.Command {
	i := experimentalaction.NewCatalogSearch(cfg,
		func(c *action.Configuration) experimentalaction.CatalogFetcher {
			return fetcher.New(c.Client)
		},
		func(c *action.Configuration) experimentalaction.CatalogContentStreamer {
			return streamer.New(c.Clientset.CoreV1())
		},
	)

	i.Logf = log.Printf

	searchOpts := experimentalaction.CatalogSearchOptions{}

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "search for catalog objects",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return search(cmd.Context(), i, searchOpts, args[0])
		},
	}

	cmd.Flags().StringVar(&searchOpts.Catalog, "catalog", "", "filter results to only be from the specified catalog")
	cmd.Flags().StringVar(&searchOpts.Schema, "schema", "", "filter results to only be FBC objects that have the specified schema")
	cmd.Flags().StringVar(&searchOpts.Package, "package", "", "filter results to only be FBC objects that belong to the specified package")

	return cmd
}

func search(ctx context.Context, searcher *experimentalaction.CatalogSearch, searchOpts experimentalaction.CatalogSearchOptions, query string) error {
	if searcher == nil {
		return errors.New("nil CatalogSearch action provided")
	}
	searchOpts.Query = query
	metas, err := searcher.Run(ctx, searchOpts)
	if err != nil {
		return fmt.Errorf("performing search: %w", err)
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
