package olmv1

import (
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime/schema"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	v1action "github.com/operator-framework/kubectl-operator/internal/pkg/v1/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

// NewExtensionInstalledGetCmd handles get commands in the form of:
// extension(s) [extension_name] - this will either list all the installed extensions
// if no extension_name has been provided or display the details of the specific
// one otherwise
func NewExtensionInstalledGetCmd(cfg *action.Configuration) *cobra.Command {
	i := v1action.NewExtensionInstalledGet(cfg)
	i.Logf = log.Printf
	var opts getOptions

	cmd := &cobra.Command{
		Use:     "extension [extension_name]",
		Aliases: []string{"extensions [extension_name]"},
		Args:    cobra.RangeArgs(0, 1),
		Short:   "Display one or many installed extensions",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 1 {
				i.ExtensionName = args[0]
			}
			if err := opts.validate(); err != nil {
				log.Fatalf("failed to parse flags: %w", err)
			}
			i.Selector = opts.ParsedSelector
			installedExtensions, err := i.Run(cmd.Context())
			if err != nil {
				log.Fatalf("failed getting installed extension(s): %w", err)
			}

			for i := range installedExtensions {
				installedExtensions[i].GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{Group: olmv1.GroupVersion.Group,
					Version: olmv1.GroupVersion.Version, Kind: olmv1.ClusterExtensionKind})
			}
			printFormattedExtensions(opts.Output, installedExtensions...)
		},
	}
	bindGetFlags(cmd.Flags(), &opts)

	return cmd
}
