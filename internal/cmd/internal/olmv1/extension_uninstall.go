package olmv1

import (
	"github.com/spf13/cobra"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	v1action "github.com/operator-framework/kubectl-operator/internal/pkg/v1/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

func NewExtensionUninstallCmd(cfg *action.Configuration) *cobra.Command {
	u := v1action.NewExtensionUninstall(cfg)
	u.Logf = log.Printf

	cmd := &cobra.Command{
		Use:   "uninstall <extension>",
		Short: "Uninstall an extension",
		Long: `Uninstall deletes the named extension object.

Warning: this command permanently deletes objects from the cluster. If the
uninstalled extension bundle contains CRDs, the CRDs will be deleted, which
cascades to the deletion of all operands.
`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			u.Package = args[0]
			if err := u.Run(cmd.Context()); err != nil {
				log.Fatalf("uninstall extension: %v", err)
			}
			log.Printf("deleted extension %q", u.Package)
		},
	}
	return cmd
}
