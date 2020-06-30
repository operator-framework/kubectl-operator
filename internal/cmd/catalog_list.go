package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/duration"

	"github.com/joelanford/kubectl-operator/internal/pkg/action"
	"github.com/joelanford/kubectl-operator/internal/pkg/log"
)

func newCatalogListCmd(cfg *action.Configuration) *cobra.Command {
	var allNamespaces bool
	l := action.NewListCatalogs(cfg)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List installed operator catalogs",
		Run: func(cmd *cobra.Command, args []string) {
			if allNamespaces {
				cfg.Namespace = v1.NamespaceAll
			}
			catalogs, err := l.Run(cmd.Context())
			if err != nil {
				log.Fatal(err)
			}

			if len(catalogs) == 0 {
				if cfg.Namespace == v1.NamespaceAll {
					log.Print("No resources found")
				} else {
					log.Printf("No resources found in %s namespace.", cfg.Namespace)
				}
				return
			}

			nsCol := ""
			if allNamespaces {
				nsCol = "\tNAMESPACE"
			}
			tw := tabwriter.NewWriter(os.Stdout, 3, 4, 2, ' ', 0)
			_, _ = fmt.Fprintf(tw, "NAME%s\tDISPLAY\tTYPE\tPUBLISHER\tAGE\n", nsCol)
			for _, cs := range catalogs {
				ns := ""
				if allNamespaces {
					ns = "\t" + cs.Namespace
				}
				age := time.Now().Sub(cs.CreationTimestamp.Time)
				_, _ = fmt.Fprintf(tw, "%s%s\t%s\t%s\t%s\t%s\n", cs.Name, ns, cs.Spec.DisplayName, cs.Spec.SourceType, cs.Spec.Publisher, duration.HumanDuration(age))
			}
			_ = tw.Flush()
		},
	}
	cmd.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "list catalogs in all namespaces")
	return cmd
}
