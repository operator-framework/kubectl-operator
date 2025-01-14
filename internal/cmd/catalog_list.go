package cmd

import (
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/duration"

	"cmp"
	catalogdv1 "github.com/operator-framework/catalogd/api/v1"
	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	internalaction "github.com/operator-framework/kubectl-operator/internal/pkg/action/v1"
	"github.com/operator-framework/kubectl-operator/pkg/action"
	"k8s.io/apimachinery/pkg/api/meta"
	"slices"
)

func newCatalogListCmd(cfg *action.Configuration) *cobra.Command {
	l := internalaction.NewCatalogList(cfg)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List cluster catalogs",
		Run: func(cmd *cobra.Command, args []string) {
			catalogs, err := l.Run(cmd.Context())
			if err != nil {
				log.Fatal(err)
			}

			if len(catalogs) == 0 {
				log.Print("No resources found")
				return
			}

			slices.SortFunc(catalogs, func(a, b catalogdv1.ClusterCatalog) int {
				return -cmp.Compare(a.Spec.Priority, b.Spec.Priority)
			})

			tw := tabwriter.NewWriter(os.Stdout, 3, 4, 2, ' ', 0)
			_, _ = fmt.Fprintf(tw, "NAME\tSERVING\tPRIORITY\tPOLL INTERVAL\tLAST UPDATE\tAGE\n")
			for _, cat := range catalogs {
				pollInterval := "Disabled"
				if cat.Spec.Source.Image != nil && cat.Spec.Source.Image.PollIntervalMinutes != nil {
					pollInterval = duration.ShortHumanDuration(time.Duration(*cat.Spec.Source.Image.PollIntervalMinutes) * time.Minute)
				}
				lastUpdate := time.Since(cat.Status.LastUnpacked.Time)
				age := time.Since(cat.CreationTimestamp.Time)
				_, _ = fmt.Fprintf(tw, "%s\t%s\t%d\t%s\t%s\t%s\n", cat.Name, servingStatus(cat), cat.Spec.Priority, pollInterval, duration.ShortHumanDuration(lastUpdate), duration.ShortHumanDuration(age))
			}
			_ = tw.Flush()
		},
	}
	return cmd
}

func servingStatus(cat catalogdv1.ClusterCatalog) metav1.ConditionStatus {
	cond := meta.FindStatusCondition(cat.Status.Conditions, catalogdv1.TypeServing)
	if cond == nil {
		return metav1.ConditionFalse
	}
	return cond.Status
}
