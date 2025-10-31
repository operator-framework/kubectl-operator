package olmv1

import (
	"github.com/spf13/cobra"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	v1action "github.com/operator-framework/kubectl-operator/internal/pkg/v1/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"
	"k8s.io/apimachinery/pkg/labels"
)

// NewExtensionInstalledGetCmd handles get commands in the form of:
// extension(s) [extension_name] - this will either list all the installed extensions
// if no extension_name has been provided or display the details of the specific
// one otherwise
func NewExtensionInstalledGetCmd(cfg *action.Configuration) *cobra.Command {
	i := v1action.NewExtensionInstalledGet(cfg)
	i.Logf = log.Printf
	extensionGetOptions := getOptions{}

	cmd := &cobra.Command{
		Use:     "extension [extension_name]",
		Aliases: []string{"extensions [extension_name]"},
		Args:    cobra.RangeArgs(0, 1),
		Short:   "Display one or many installed extensions",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 1 {
				i.ExtensionName = args[0]
			}
			var err error
			if len(extensionGetOptions.Selector) > 0 {
				i.Selector, err = labels.Parse(extensionGetOptions.Selector)
				if err != nil {
					log.Fatalf("unable to parse selector %s: %v", extensionGetOptions.Selector, err)
				}
			}
			installedExtensions, err := i.Run(cmd.Context())
			if err != nil {
				log.Fatalf("failed getting installed extension(s): %v", err)
			}

			printFormattedExtensions(extensionGetOptions.Output, installedExtensions...)
		},
	}
	bindGetFlags(cmd.Flags(), &extensionGetOptions)

	return cmd
}
