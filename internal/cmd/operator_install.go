package cmd

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	internalaction "github.com/operator-framework/kubectl-operator/internal/pkg/action/v1"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

func newExtensionInstallCmd(cfg *action.Configuration) *cobra.Command {
	i := internalaction.NewOperatorInstall(cfg)
	i.Logf = log.Printf

	cmd := &cobra.Command{
		Use:   "install <operator>",
		Short: "Install an operator",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			i.Package = args[0]
			i.Namespace = internalaction.OperatorInstallNamespaceConfig{
				Name: cfg.Namespace,
			}
			clusterExtension, err := i.Run(cmd.Context())
			if err != nil {
				log.Fatalf("failed to install cluster extension: %v", err)
			}
			log.Printf("cluster extension %q installed; installed bundle is %q", i.Package, clusterExtension.Status.Install.Bundle.Name)
		},
	}
	bindOperatorInstallFlags(cmd.Flags(), i)

	return cmd
}

func bindOperatorInstallFlags(fs *pflag.FlagSet, i *internalaction.OperatorInstall) {
	fs.StringSliceVarP(&i.Channels, "channels", "c", []string{}, "upgrade channels from which to resolve bundles")
	fs.StringVarP(&i.Version, "version", "v", "", "version (or version range) from which to resolve bundles")
	fs.DurationVar(&i.CleanupTimeout, "cleanup-timeout", time.Minute, "the amount of time to wait before cancelling cleanup")
	fs.BoolVarP(&i.UnsafeCreateClusterRoleBinding, "unsafe-create-cluster-role-binding", "X", false, "create a cluster-admin ClusterRoleBinding for the extension installation")
	fs.StringVarP(&i.ServiceAccount, "service-account", "s", "default", "service account to use for the extension installation")
}
