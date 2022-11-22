package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	internalaction "github.com/operator-framework/kubectl-operator/internal/pkg/action"
	"github.com/operator-framework/kubectl-operator/internal/pkg/operator"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

var (
	// output helpers for the describe subcommand
	catHdr  = asHeader("Catalog")
	pkgHdr  = asHeader("Package")
	repoHdr = asHeader("Repository")
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

			// prepare what we want to print to the console
			out := make([]string, 0)

			// The PackageManifest API is a bit unusual, as it is namespace-scoped, but has the concept of a "Global" package.
			// For Global catalogs, the PackageManifest is available to each namespace.
			// The -n global parameter is ignored by this command
			// Instead, the -c argument accepts a NamespacedName: [namespace][/][catalog-name]
			//   All catalogs:                     /
			//   All catalogs in one namespace:    mynamespace/
			//   All catalogs with the same name:  /mycatalog
			//                                     mycatalog
			//   Specific catalog:                 mynamespace/mycatalog

			// If no namespace is explicitly selected, show the package in all available catalogs visible to the user.
			// If a namespace is explicitly selected, show the package scoped to the catalogs in the specified namespace.

			for _, pm := range pms {

				globalScope := true
				if pm.Namespace == pm.Status.CatalogSourceNamespace {
					globalScope = false
				}

				// catalog
				out = append(out, catHdr)

				if globalScope {
					out = append(out,
						"Scope: Global\n",
					)
				} else {
					out = append(out,
						"Scope: Namespaced\n",
						fmt.Sprintf("Namespace: %s\n", pm.Labels["catalog-namespace"]),
					)
				}
				out = append(out,
					fmt.Sprintf("Name: %s\n", pm.Status.CatalogSource),
					fmt.Sprintf("Display Name: %s\n", pm.Status.CatalogSourceDisplayName),
					fmt.Sprintf("Publisher: %s\n\n", pm.Status.CatalogSourcePublisher),
				)

				pc, err := pm.GetChannel(channel)
				if err != nil {
					// the requested channel doesn't exist
					log.Fatal(err)
				}

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
					// available channels
					chHdr+fmt.Sprintf("%s\n\n",
						strings.Join(getAvailableChannelsWithMarkers(*pc, pm), "\n")),
					// install modes
					imHdr+fmt.Sprintf("%s\n\n",
						strings.Join(pc.GetSupportedInstallModes().List(), "\n")),
					// description
					sdHdr+fmt.Sprintf("%s\n",
						pc.CurrentCSVDesc.Annotations[descAnnot]),
				)

				// if the user requested a long description, add it to the output as well
				if longDescription {
					out = append(out,
						"\n"+ldHdr+pm.Status.Channels[0].CurrentCSVDesc.LongDescription)
				}

				out = append(out, "\n")

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
