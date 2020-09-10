package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	"github.com/operator-framework/kubectl-operator/internal/pkg/action"
)

func newOperatorDescribeCmd(cfg *action.Configuration) *cobra.Command {
	i := action.NewOperatorDescribe(cfg)
	i.Logf = log.Printf

	cmd := &cobra.Command{
		Use:   "describe <operator>",
		Short: "Describe an operator",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			pkgHdr := asHeader("Package")
			repoHdr := asHeader("Repository")
			catHdr := asHeader("Catalog")
			chHdr := asHeader("Channels")
			imHdr := asHeader("Install Modes")
			sdHdr := asHeader("Description")
			ldHdr := asHeader("Long Description")

			repoAnnot := "repository"
			descAnnot := "description"

			// the operator to show details about, provided by the user
			i.Package = args[0]

			ctx, cancel := context.WithTimeout(cmd.Context(), i.ShowTimeout)
			defer cancel()

			// Find the package manifest and package channel for the operator
			pm, pc, err := i.Run(ctx)
			if err != nil {
				log.Fatalf("failed to find operator: %v", err)
			}

			// prepare what we want to print to the console
			out := make([]string, 0)

			// Starting adding data to our output.
			out = append(out,
				// package
				pkgHdr+fmt.Sprintf("%s %s (by %s)\n\n",
					pc.CurrentCSVDesc.DisplayName,
					pc.CurrentCSVDesc.Version,
					pc.CurrentCSVDesc.Provider.Name),
				// repo
				repoHdr+fmt.Sprintf("%s\n\n",
					pc.CurrentCSVDesc.Annotations[repoAnnot]),
				// catalog
				catHdr+fmt.Sprintf("%s\n\n", pm.Status.CatalogSourceDisplayName),
				// available channels
				chHdr+fmt.Sprintf("%s\n\n",
					asNewlineDelimString(i.GetAvailableChannels(pm))),
				// install modes
				imHdr+fmt.Sprintf("%s\n\n",
					asNewlineDelimString(i.GetSupportedInstallModes(pc))),
				// description
				sdHdr+fmt.Sprintf("%s\n",
					pc.CurrentCSVDesc.Annotations[descAnnot]),
			)

			// if the user requested a long description, add it to the output
			if i.LongDescription {
				out = append(out,
					"\n"+ldHdr+pm.Status.Channels[0].CurrentCSVDesc.LongDescription)
			}

			// finally, print operator information to the console
			for _, v := range out {
				fmt.Print(v)
			}

		},
	}

	// add flags to the flagset for this command.
	cmd.Flags().StringVarP(&i.Channel, "channel", "c", "", "channel")
	cmd.Flags().BoolVarP(&i.LongDescription, "with-long-description", "L", false, "long description")
	cmd.Flags().DurationVarP(&i.ShowTimeout, "timeout", "t", time.Minute, "the amount of time to wait before cancelling the show request")
	return cmd
}

// asNewlineDelimString returns a string slice as a single string
// separated by newlines
func asNewlineDelimString(stringItems []string) string {
	var res string
	for _, v := range stringItems {
		if res != "" {
			res = fmt.Sprintf("%s\n%s", res, v)
			continue
		}

		res = v
	}
	return res
}

// asHeader returns the string with "header bars" for displaying in
// plain text cases.
func asHeader(s string) string {
	return fmt.Sprintf("== %s ==\n", s)
}
