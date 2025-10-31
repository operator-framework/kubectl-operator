package olmv1

import (
	"github.com/spf13/cobra"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	v1action "github.com/operator-framework/kubectl-operator/internal/pkg/v1/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"
	"k8s.io/apimachinery/pkg/labels"
)

// NewCatalogInstalledGetCmd handles get commands in the form of:
// catalog(s) [catalog_name] - this will either list all the installed operators
// if no catalog_name has been provided or display the details of the specific
// one otherwise
func NewCatalogInstalledGetCmd(cfg *action.Configuration) *cobra.Command {
	i := v1action.NewCatalogInstalledGet(cfg)
	i.Logf = log.Printf
	catalogGetOptions := getOptions{}

	cmd := &cobra.Command{
		Use:     "catalog [catalog_name]",
		Aliases: []string{"catalogs"},
		Args:    cobra.RangeArgs(0, 1),
		Short:   "Display one or many installed catalogs",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 1 {
				i.CatalogName = args[0]
			}
			var err error
			switch catalogGetOptions.Output {
			case "yaml", "json", "":
			default:
				log.Fatalf("unrecognized output format %s", catalogGetOptions.Output)
			}
			if len(catalogGetOptions.Selector) > 0 {
				i.Selector, err = labels.Parse(catalogGetOptions.Selector)
				if err != nil {
					log.Fatalf("unable to parse selector %s: %v", catalogGetOptions.Selector, err)
				}
			}
			installedCatalogs, err := i.Run(cmd.Context())
			if err != nil {
				log.Fatalf("failed getting installed catalog(s): %v", err)
			}
			printFormattedCatalogs(catalogGetOptions.Output, installedCatalogs...)
		},
	}
	bindGetFlags(cmd.Flags(), &catalogGetOptions)

	return cmd
}
