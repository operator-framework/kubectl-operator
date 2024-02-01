package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/operator-framework/kubectl-operator/internal/pkg/catalogd/fetcher"
	"github.com/operator-framework/kubectl-operator/internal/pkg/catalogd/streamer"
	experimentalaction "github.com/operator-framework/kubectl-operator/internal/pkg/experimental/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"
	"sigs.k8s.io/yaml"

	"github.com/spf13/cobra"
)

type InspectCommandOptions struct {
	experimentalaction.CatalogInspectOptions
	Output string
}

func NewInspectCommand(cfg *action.Configuration) *cobra.Command {
	i := experimentalaction.NewCatalogInspect(cfg,
		func(c *action.Configuration) experimentalaction.CatalogFetcher {
			return fetcher.New(c.Client)
		},
		func(c *action.Configuration) experimentalaction.CatalogContentStreamer {
			return streamer.New(c.Clientset.CoreV1())
		},
	)
	i.Logf = log.Printf

	inspectOpts := InspectCommandOptions{}

	cmd := &cobra.Command{
		Use:   "inspect [name]",
		Short: "inspect catalog objects",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return inspect(cmd.Context(), i, inspectOpts.CatalogInspectOptions, args[0], inspectOpts.Output)
		},
	}

	cmd.Flags().StringVar(&inspectOpts.Catalog, "catalog", "", "filter results to only be from the specified catalog")
	cmd.Flags().StringVar(&inspectOpts.Schema, "schema", "", "filter results to only be FBC objects that have the specified schema")
	cmd.Flags().StringVar(&inspectOpts.Package, "package", "", "filter results to only be FBC objects that belong to the specified package")
	cmd.Flags().StringVar(&inspectOpts.Output, "output", "json", "the format in which output should be. One of [json, yaml]")

	return cmd
}

func inspect(ctx context.Context, inspector *experimentalaction.CatalogInspect, inspectOpts experimentalaction.CatalogInspectOptions, name, format string) error {
	if inspector == nil {
		return errors.New("nil CatalogInspect action provided")
	}
	metas, err := inspector.Run(ctx, inspectOpts)
	if err != nil {
		return fmt.Errorf("performing inspect: %w", err)
	}

	for _, meta := range metas {
		var outBytes []byte
		var err error
		switch format {
		case "json":
			outBytes, err = json.MarshalIndent(meta.Meta, "", " ")
			if err != nil {
				return fmt.Errorf("marshalling output: %w", err)
			}
		case "yaml":
			outBytes, err = yaml.Marshal(meta.Meta)
			if err != nil {
				return fmt.Errorf("marshalling output: %w", err)
			}
		}

		fmt.Println(string(outBytes))
	}
	return nil
}
