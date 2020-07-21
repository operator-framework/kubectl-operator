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
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
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
			_, _ = fmt.Fprintf(tw, "NAME\tCATALOG\tAGE\n")
			for _, op := range operators {
				age := time.Now().Sub(op.CreationTimestamp.Time)
				_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\n", op.Name, op.Status.CatalogSourceDisplayName, duration.HumanDuration(age))
			}
			_ = tw.Flush()
		},
	}
	l.BindFlags(cmd.Flags())
	return cmd
}
