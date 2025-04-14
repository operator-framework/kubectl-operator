package olmv1

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	v1action "github.com/operator-framework/kubectl-operator/internal/pkg/v1/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

func NewExtensionInstallCmd(cfg *action.Configuration) *cobra.Command {
	i := v1action.NewExtensionInstall(cfg)
	i.Logf = log.Printf

	cmd := &cobra.Command{
		Use:   "extension <extension_name>",
		Short: "Install an extension",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			i.ExtensionName = args[0]
			_, err := i.Run(cmd.Context())
			if err != nil {
				log.Fatalf("failed to install extension: %v", err)
			}
			log.Printf("extension %q created", i.ExtensionName)
		},
	}
	bindOperatorInstallFlags(cmd.Flags(), i)

	return cmd
}

func bindOperatorInstallFlags(fs *pflag.FlagSet, i *v1action.ExtensionInstall) {
	fs.StringVarP(&i.Namespace.Name, "namespace", "n", "", "namespace to install the operator in")
	fs.StringVarP(&i.PackageName, "package-name", "p", "", "package name of the operator to install")
	fs.StringSliceVarP(&i.Channels, "channels", "c", []string{}, "channels which would be to used for getting updates e.g --channels \"stable,dev-preview,preview\"")
	fs.StringVarP(&i.Version, "version", "v", "", "version (or version range) from which to resolve bundles")
	fs.StringVarP(&i.ServiceAccount, "service-account", "s", "default", "service account name to use for the extension installation")
	fs.DurationVarP(&i.CleanupTimeout, "cleanup-timeout", "d", time.Minute, "the amount of time to wait before cancelling cleanup after a failed creation attempt")
}
