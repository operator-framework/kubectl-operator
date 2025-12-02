package olmv1

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime/schema"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	v1action "github.com/operator-framework/kubectl-operator/internal/pkg/v1/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

type extensionDeleteOptions struct {
	dryRunOptions
}

// NewExtensionDeleteCmd deletes either a specific extension by name
// or all extensions on cluster.
func NewExtensionDeleteCmd(cfg *action.Configuration) *cobra.Command {
	i := v1action.NewExtensionDelete(cfg)
	i.Logf = log.Printf
	var opts extensionDeleteOptions

	cmd := &cobra.Command{
		Use:     "extension [extension_name]",
		Aliases: []string{"extensions [extension_name]"},
		Short:   "Delete either a single or all of the existing extensions",
		Long: `Warning: Permanently deletes the named cluster extension object.
		If the extension contains CRDs, the CRDs will be deleted, which
		 cascades to the deletion of all operands.`,
		Args: cobra.RangeArgs(0, 1),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				if i.DeleteAll {
					log.Fatalf("failed to delete extension: cannot specify both --all and an extension name")
				}
				i.ExtensionName = args[0]
			}
			if err := opts.validate(); err != nil {
				log.Fatalf("failed to parse flags: %w", err)
			}
			i.DryRun = opts.DryRun
			i.Output = opts.Output
			extensions, err := i.Run(cmd.Context())
			if err != nil {
				log.Fatalf("failed to delete extension: %w", err)
			}
			if len(i.DryRun) == 0 {
				for _, e := range extensions {
					log.Printf("extension %q deleted", e.Name)
				}
				return
			}
			if len(i.Output) == 0 {
				for _, e := range extensions {
					log.Printf("extension %q deleted (dry run)", e.Name)
				}
				return
			}

			for _, e := range extensions {
				e.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{Group: olmv1.GroupVersion.Group,
					Version: olmv1.GroupVersion.Version, Kind: olmv1.ClusterExtensionKind})
			}
			printFormattedExtensions(i.Output, extensions...)
		},
	}
	bindExtensionDeleteFlags(cmd.Flags(), i)
	bindDryRunFlags(cmd.Flags(), &opts.dryRunOptions)

	return cmd
}

func bindExtensionDeleteFlags(fs *pflag.FlagSet, e *v1action.ExtensionDeletion) {
	fs.BoolVarP(&e.DeleteAll, "all", "a", false, "delete all extensions.")
}
