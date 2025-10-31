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

func NewExtensionDeleteCmd(cfg *action.Configuration) *cobra.Command {
	e := v1action.NewExtensionDelete(cfg)
	e.Logf = log.Printf

	cmd := &cobra.Command{
		Use:     "extension [extension_name]",
		Aliases: []string{"extensions [extension_name]"},
		Short:   "Delete an extension",
		Long: `Warning: Permanently deletes the named cluster extension object.
		If the extension contains CRDs, the CRDs will be deleted, which
		 cascades to the deletion of all operands.`,
		Args: cobra.RangeArgs(0, 1),
		Run: func(cmd *cobra.Command, args []string) {
			if len(e.DryRun) > 0 && e.DryRun != v1action.DryRunAll {
				log.Fatalf("invalid value for `--dry-run` %s, must be one of (%s)\n", e.DryRun, v1action.DryRunAll)
			}
			if len(args) != 0 {
				e.ExtensionName = args[0]
			}
			extensions, err := e.Run(cmd.Context())
			if err != nil {
				log.Fatalf("failed to delete extension: %v", err)
			}
			if len(e.DryRun) == 0 {
				for _, extn := range extensions {
					log.Printf("extension %s deleted", extn.Name)
				}
				return
			}
			if len(e.Output) == 0 {
				for _, extn := range extensions {
					log.Printf("extension %s deleted(dry run)\n", extn.Name)
				}
				return
			}

			for _, i := range extensions {
				i.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{Group: olmv1.GroupVersion.Group,
					Version: olmv1.GroupVersion.Version, Kind: olmv1.ClusterExtensionKind})
			}
			printFormattedExtensions(e.Output, extensions...)
		},
	}
	bindExtensionDeleteFlags(cmd.Flags(), e)
	return cmd
}

func bindExtensionDeleteFlags(fs *pflag.FlagSet, e *v1action.ExtensionDeletion) {
	fs.BoolVarP(&e.DeleteAll, "all", "a", false, "delete all extensions")
	fs.StringVar(&e.DryRun, "dry-run", "", "Display the object that would be sent on a request without applying it. One of: (All)")
	fs.StringVarP(&e.Output, "output", "o", "", "Output format for dry-run manifests. One of: (json, yaml)")
}
