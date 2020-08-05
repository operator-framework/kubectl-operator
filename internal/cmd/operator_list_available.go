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

func newOperatorListAvailableCmd(cfg *action.Configuration) *cobra.Command {
	l := action.NewOperatorListAvailable(cfg)
	cmd := &cobra.Command{
		Use:   "list-available",
		Short: "List operators available to be installed",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 1 {
				l.Package = args[0]
			}

			operators, err := l.Run(cmd.Context())
			if err != nil {
				log.Fatal(err)
			}

			if len(operators) == 0 {
				if cfg.Namespace == v1.NamespaceAll {
					log.Print("No resources found")
				} else {
					log.Printf("No resources found in %s namespace.\n", cfg.Namespace)
				}
				return
			}

			sort.SliceStable(operators, func(i, j int) bool {
				return strings.Compare(operators[i].Name, operators[j].Name) < 0
			})

			tw := tabwriter.NewWriter(os.Stdout, 3, 4, 2, ' ', 0)
			_, _ = fmt.Fprintf(tw, "NAME\tCATALOG\tCHANNEL\tLATEST CSV\tAGE\n")
			for _, op := range operators {
				age := time.Since(op.CreationTimestamp.Time)
				for _, ch := range op.Status.Channels {
					_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", op.Name, op.Status.CatalogSourceDisplayName, ch.Name, ch.CurrentCSV, duration.HumanDuration(age))
				}
			}
			_ = tw.Flush()
		},
	}
	l.BindFlags(cmd.Flags())
	return cmd
}
