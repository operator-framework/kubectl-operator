package olmv1

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	v1action "github.com/operator-framework/kubectl-operator/internal/pkg/v1/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type extensionDeleteOptions struct {
	dryRunOptions
}

func NewExtensionDeleteCmd(cfg *action.Configuration) *cobra.Command {
	i := v1action.NewExtensionDelete(cfg)
	i.Logf = log.Printf
	var opts extensionDeleteOptions

	cmd := &cobra.Command{
		Use:     "extension [extension_name]",
		Aliases: []string{"extensions [extension_name]"},
		Short:   "Delete an extension",
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
				log.Fatalf("failed to parse flags: %s", err.Error())
			}
			i.DryRun = opts.DryRun
			i.Output = opts.Output
			extensions, err := i.Run(cmd.Context())
			if err != nil {
				log.Fatalf("failed to delete extension: %v", err)
			}
			if len(i.DryRun) == 0 {
				for _, extn := range extensions {
					log.Printf("extension %s deleted", extn.Name)
				}
				return
			}
			if len(i.Output) == 0 {
				for _, extn := range extensions {
					log.Printf("extension %s deleted (dry run)\n", extn.Name)
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
	fs.BoolVarP(&e.DeleteAll, "all", "a", false, "delete all extensions")
}
