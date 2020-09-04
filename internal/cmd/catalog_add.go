package cmd

import (
	"context"
	"io/ioutil"
	"time"

	"github.com/operator-framework/operator-registry/pkg/image/containerdregistry"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	"github.com/operator-framework/kubectl-operator/internal/pkg/action"
)

func newCatalogAddCmd(cfg *action.Configuration) *cobra.Command {
	var timeout time.Duration
	a := action.NewCatalogAdd(cfg)
	a.Logf = log.Printf

	cmd := &cobra.Command{
		Use:   "add <name> <index_image>",
		Short: "Add an operator catalog",
		Args:  cobra.ExactArgs(2),
		PreRun: func(cmd *cobra.Command, args []string) {
			regLogger := logrus.New()
			regLogger.SetOutput(ioutil.Discard)
			a.RegistryOptions = []containerdregistry.RegistryOption{
				containerdregistry.WithLog(logrus.NewEntry(regLogger)),
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()

			a.CatalogSourceName = args[0]
			a.IndexImage = args[1]

			cs, err := a.Run(ctx)
			if err != nil {
				log.Fatalf("failed to add catalog: %v", err)
			}
			log.Printf("created catalogsource %q\n", cs.Name)
		},
	}
	bindCatalogAddFlags(cmd.Flags(), a)
	cmd.Flags().DurationVarP(&timeout, "timeout", "t", time.Minute, "the amount of time to wait before cancelling the catalog addition")

	return cmd
}

func bindCatalogAddFlags(fs *pflag.FlagSet, a *action.CatalogAdd) {
	fs.StringVarP(&a.DisplayName, "display-name", "d", "", "display name of the index")
	fs.StringVarP(&a.Publisher, "publisher", "p", "", "publisher of the index")
	fs.DurationVar(&a.CleanupTimeout, "cleanup-timeout", time.Minute, "the amount to time to wait before cancelling cleanup")

	fs.StringArrayVarP(&a.InjectBundles, "inject-bundles", "b", nil, "inject extra bundles into the index at runtime")
	fs.StringVarP(&a.InjectBundleMode, "inject-bundle-mode", "m", "", "mode to use to inject bundles")
	_ = fs.MarkHidden("inject-bundle-mode")
}
