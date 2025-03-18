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
		Use:   "extension <extension-name>",
		Short: "Delete an extension",
		Long: `Warning: Permanently deletes the named cluster extension object.
		If the extension contains CRDs, the CRDs will be deleted, which
		 cascades to the deletion of all operands.`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			e.ExtensionName = args[0]
			if err := e.Run(cmd.Context()); err != nil {
				log.Fatalf("delete extension: %v", err)
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
