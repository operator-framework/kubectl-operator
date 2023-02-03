package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	internalaction "github.com/operator-framework/kubectl-operator/internal/pkg/action"
	"github.com/operator-framework/kubectl-operator/internal/pkg/operator"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

var (
	// output helpers for the describe subcommand
	pkgHdr  = asHeader("Package")
	repoHdr = asHeader("Repository")
	catHdr  = asHeader("Catalog")
	chHdr   = asHeader("Channels")
	imHdr   = asHeader("Install Modes")
	sdHdr   = asHeader("Description")
	ldHdr   = asHeader("Long Description")

	repoAnnot = "repository"
	descAnnot = "description"
)

func newOperatorDescribeCmd(cfg *action.Configuration) *cobra.Command {
	l := internalaction.NewOperatorListAvailable(cfg)
	// receivers for cmdline flags
	var channel string
	var longDescription bool

	cmd := &cobra.Command{
		Use:   "describe <operator>",
		Short: "Describe an operator",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// the operator to show details about, provided by the user
			l.Package = args[0]

			// Find the package manifest and package channel for the operator
			pms, err := l.Run(cmd.Context())
			if err != nil {
				log.Fatal(err)
			}

			// we only expect one item because describe always searches
			// for a specific operator by name
			pm := pms[0]

			pc, err := pm.GetChannel(channel)
			if err != nil {
				// the requested channel doesn't exist
				log.Fatal(err)
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
					strings.Join(getAvailableChannelsWithMarkers(*pc, pm), "\n")),
				// install modes
				imHdr+fmt.Sprintf("%s\n\n",
					strings.Join(sets.List[string](pc.GetSupportedInstallModes()), "\n")),
				// description
				sdHdr+fmt.Sprintf("%s\n",
					pc.CurrentCSVDesc.Annotations[descAnnot]),
			)

			// if the user requested a long description, add it to the output as well
			if longDescription {
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
	bindOperatorListAvailableFlags(cmd.Flags(), l)
	cmd.Flags().StringVarP(&channel, "channel", "C", "", "package channel to describe")
	cmd.Flags().BoolVarP(&longDescription, "with-long-description", "L", false, "include long description")

	return cmd
}

// asHeader returns the string with "header bars" for displaying in
// plain text cases.
func asHeader(s string) string {
	return fmt.Sprintf("== %s ==\n", s)
}

// getAvailableChannelsWithMarkers parses all available package channels for a package manifest
// and returns those channel names with indicators for pretty-printing whether they are shown
// or the default channel
func getAvailableChannelsWithMarkers(channel operator.PackageChannel, pm operator.PackageManifest) []string {
	channels := make([]string, len(pm.Status.Channels))
	for i, ch := range pm.Status.Channels {
		n := ch.Name
		if ch.IsDefaultChannel(pm.PackageManifest) {
			n += " (default)"
		}
		if channel.Name == ch.Name {
			n += " (shown)"
		}
		channels[i] = n
	}

	return channels
}
