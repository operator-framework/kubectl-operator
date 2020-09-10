package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	operatorsv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/apis/operators/v1"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	"github.com/operator-framework/kubectl-operator/internal/pkg/action"
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
	l := action.NewOperatorListAvailable(cfg)
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
				log.Fatalf("failed to find operator: %v", err)
			}

			// we only expect one item because describe always searches
			// for a specific operator by name
			pm := &pms[0]

			// If the user asked for a channel, look for that
			if channel == "" {
				channel = pm.Status.DefaultChannel
			}

			pc, err := getPackageChannel(channel, pm)
			if err != nil {
				// the requested channel doesn't exist
				log.Fatalf("failed to find channel for operator: %v", err)
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
					asNewlineDelimString(getAvailableChannelsWithMarkers(channel, pm))),
				// install modes
				imHdr+fmt.Sprintf("%s\n\n",
					asNewlineDelimString(getSupportedInstallModes(pc))),
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
	cmd.Flags().StringVarP(&channel, "channel", "c", "", "channel")
	cmd.Flags().BoolVarP(&longDescription, "with-long-description", "L", false, "long description")

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

// getPackageChannel returns the package channel specified, or the default if none was specified.
func getPackageChannel(channel string, pm *operatorsv1.PackageManifest) (*operatorsv1.PackageChannel, error) {
	var packageChannel *operatorsv1.PackageChannel
	for _, ch := range pm.Status.Channels {
		ch := ch
		if ch.Name == channel {
			packageChannel = &ch
		}
	}
	if packageChannel == nil {
		return nil, fmt.Errorf("channel %q does not exist for package %q", channel, pm.GetName())
	}
	return packageChannel, nil
}

// GetSupportedInstallModes returns a string slice representation of install mode
// objects the operator supports.
func getSupportedInstallModes(pc *operatorsv1.PackageChannel) []string {
	supportedInstallModes := make([]string, 1)
	for _, imode := range pc.CurrentCSVDesc.InstallModes {
		if imode.Supported {
			supportedInstallModes = append(supportedInstallModes, string(imode.Type))
		}
	}

	return supportedInstallModes
}

// getAvailableChannelsWithMarkers parses all available package channels for a package manifest
// and returns those channel names with indicators for pretty-printing whether they are shown
// or the default channel
func getAvailableChannelsWithMarkers(channel string, pm *operatorsv1.PackageManifest) []string {
	channels := make([]string, len(pm.Status.Channels))
	for i, ch := range pm.Status.Channels {
		n := ch.Name
		if ch.IsDefaultChannel(*pm) {
			n += " (default)"
		}
		if channel == ch.Name {
			n += " (shown)"
		}
		channels[i] = n
	}

	return channels
}
