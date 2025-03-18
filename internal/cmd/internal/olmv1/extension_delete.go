package olmv1

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	v1action "github.com/operator-framework/kubectl-operator/internal/pkg/v1/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"
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
			if len(args) == 0 {
				extensions, err := e.Run(cmd.Context())
				if err != nil {
					log.Fatalf("failed deleting extension: %v", err)
				}
				for _, extn := range extensions {
					log.Printf("extension %q deleted", extn)
				}

				return
			}
			e.ExtensionName = args[0]
			_, errs := e.Run(cmd.Context())
			if errs != nil {
				log.Fatalf("delete extension: %v", errs)
			}
			log.Printf("deleted extension %q", e.ExtensionName)
		},
	}
	bindExtensionDeleteFlags(cmd.Flags(), e)
	return cmd
}

func bindExtensionDeleteFlags(fs *pflag.FlagSet, e *v1action.ExtensionDeletion) {
	fs.BoolVarP(&e.DeleteAll, "all", "a", false, "delete all extensions")
}
