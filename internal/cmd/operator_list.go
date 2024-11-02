package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/duration"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	internalaction "github.com/operator-framework/kubectl-operator/internal/pkg/action/v1"
	"github.com/operator-framework/kubectl-operator/pkg/action"
	ocv1 "github.com/operator-framework/operator-controller/api/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"slices"
)

func newExtensionListCmd(cfg *action.Configuration) *cobra.Command {
	l := internalaction.NewOperatorList(cfg)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List cluster extensions",
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			clusterExtensions, err := l.Run(cmd.Context())
			if err != nil {
				log.Fatalf("list operators: %v", err)
			}

			if len(clusterExtensions) == 0 {
				log.Print("No resources found")
				return
			}

			slices.SortFunc(clusterExtensions, func(a, b ocv1.ClusterExtension) int {
				return strings.Compare(a.Status.Install.Bundle.Name, b.Status.Install.Bundle.Name)
			})
			tw := tabwriter.NewWriter(os.Stdout, 3, 4, 2, ' ', 0)
			_, _ = fmt.Fprintf(tw, "NAME\tNAMESPACE\tINSTALLED BUNDLE\tAT DESIRED STATE\tAGE\n")
			for _, ce := range clusterExtensions {
				installedBundle := "(not installed)"
				if meta.IsStatusConditionPresentAndEqual(ce.Status.Conditions, ocv1.TypeInstalled, metav1.ConditionTrue) {
					installedBundle = ce.Status.Install.Bundle.Name
				}
				atDesiredState := "False"
				progressing := meta.FindStatusCondition(ce.Status.Conditions, ocv1.TypeProgressing)
				if progressing != nil && progressing.Status == metav1.ConditionTrue && progressing.Reason == ocv1.ReasonSucceeded {
					atDesiredState = "True"
				}

				age := time.Since(ce.CreationTimestamp.Time)
				_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", ce.Name, ce.Spec.Namespace, installedBundle, atDesiredState, duration.HumanDuration(age))
			}
			_ = tw.Flush()
		},
	}
	return cmd
}
