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
	allNamespaces := new(bool)

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
			crLister.AllNamespaces = *allNamespaces

			op, err := crLister.FindOperator(cmd.Context())
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("found operator %s", op.Name)

			crds, err := crLister.Unzip(cmd.Context(), op)
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("found owned crds #%v", crds)

			list, err := crLister.List(cmd.Context(), crds)
			if err != nil {
				log.Fatal(err)
			}

			// pretty-print if no output specified
			if *output == "" {
				tw := tabwriter.NewWriter(os.Stdout, 3, 4, 2, ' ', 0)
				_, _ = fmt.Fprintf(tw, "NAME\tNAMESPACE\tKIND\tAPIVERSION\tAGE\n")
				for _, cr := range list.Items {
					age := time.Since(cr.CreationTimestamp.Time)
					_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s", cr.Name, cr.Namespace, cr.Kind, cr.APIVersion, duration.HumanDuration(age))
				}
				_ = tw.Flush()
			} else if *output == "json" {
				for _, cr := range list.Items {
					fmt.Printf("#%v", cr.String())
				}
			} else if *output == "yaml" {
				//TODO
			}
		},
	}


	bindOperatorListCustomResourcesFlags(cmd.Flags(), allNamespaces, output)
	return cmd
}

func bindOperatorListCustomResourcesFlags(fs *pflag.FlagSet, allNamespaces *bool, output *string) {
	fs.BoolVarP(allNamespaces, "all-namespaces", "A", false, "list operators in all namespaces")
	fs.StringVarP(output, "output", "o", "", "Determines format for list output. One of json or yaml.")
}

func notValid(output *string) bool {
	a := *output
	if  a == "" || a == "json" || a == "yaml" {
		return true
	}
	return false
}
