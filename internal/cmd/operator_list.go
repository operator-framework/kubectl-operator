package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/duration"

	"github.com/joelanford/kubectl-operator/internal/pkg/action"
	"github.com/joelanford/kubectl-operator/internal/pkg/log"
)

func newOperatorListCmd(cfg *action.Configuration) *cobra.Command {
	var allNamespaces bool
	l := action.NewOperatorList(cfg)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List installed operators",
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			if allNamespaces {
				cfg.Namespace = v1.NamespaceAll
			}
			subs, err := l.Run(cmd.Context())
			if err != nil {
				log.Fatalf("list operators: %v", err)
			}

			if len(subs) == 0 {
				if cfg.Namespace == v1.NamespaceAll {
					log.Print("No resources found")
				} else {
					log.Printf("No resources found in %s namespace.", cfg.Namespace)
				}
				return
			}

			sort.SliceStable(subs, func(i, j int) bool {
				return strings.Compare(subs[i].Spec.Package, subs[j].Spec.Package) < 0
			})
			nsCol := ""
			if allNamespaces {
				nsCol = "\tNAMESPACE"
			}
			tw := tabwriter.NewWriter(os.Stdout, 3, 4, 2, ' ', 0)
			_, _ = fmt.Fprintf(tw, "PACKAGE%s\tSUBSCRIPTION\tINSTALLED CSV\tCURRENT CSV\tSTATUS\tAGE\n", nsCol)
			for _, sub := range subs {
				ns := ""
				if allNamespaces {
					ns = "\t" + sub.Namespace
				}
				age := time.Since(sub.CreationTimestamp.Time)
				_, _ = fmt.Fprintf(tw, "%s%s\t%s\t%s\t%s\t%s\t%s\n", sub.Spec.Package, ns, sub.Name, sub.Status.InstalledCSV, sub.Status.CurrentCSV, sub.Status.State, duration.HumanDuration(age))
			}
			_ = tw.Flush()

		},
	}
	cmd.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "list operators in all namespaces")
	return cmd
}
