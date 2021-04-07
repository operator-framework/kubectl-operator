package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/duration"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	"github.com/operator-framework/kubectl-operator/internal/pkg/action"
)

func newOperatorListCustomResourcesCmd(cfg *action.Configuration) *cobra.Command {
	crLister := action.NewOperatorListCustomResources(cfg)
	output := new(string)

	cmd := &cobra.Command{
		Use:     "list-custom-resources <operator>",
		Aliases: []string{"list-crs"},
		Short:   "List custom resources for an operator",
		Long:    "List all custom resources on-cluster associated with a particular operator",
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				log.Fatal("did not provide operator or package name")
			}
			if notValid(output) {
				log.Fatal("invalid output type provided")
			}

			crLister.PackageName = args[0]

			list, err := crLister.Run(cmd.Context())
			if err != nil {
				log.Fatal(err)
			}

			// pretty-print if no output specified
			if *output == "" {
				tw := tabwriter.NewWriter(os.Stdout, 3, 4, 2, ' ', 0)
				_, _ = fmt.Fprintf(tw, "NAME\tNAMESPACE\tKIND\tAPIVERSION\tAGE\n")
				for _, cr := range list.Items {
					age := time.Since(cr.GetCreationTimestamp().Time)
					_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s", cr.GetName(), cr.GetNamespace(), cr.GetKind(), cr.GetAPIVersion(), duration.HumanDuration(age))
				}
				_ = tw.Flush()
			} else if *output == "json" {
				for _, cr := range list.Items {
					j, _ := cr.MarshalJSON()
					fmt.Printf("#%v", string(j))
				}
			} else if *output == "yaml" {
				//TODO
			}
		},
	}

	bindOperatorListCustomResourcesFlags(cmd.Flags(), output)
	return cmd
}

func bindOperatorListCustomResourcesFlags(fs *pflag.FlagSet, output *string) {
	fs.StringVarP(output, "output", "o", "", "Determines format for list output. One of json or yaml.")
}

func notValid(output *string) bool {
	a := *output
	if a == "" || a == "json" || a == "yaml" {
		return true
	}
	return false
}
