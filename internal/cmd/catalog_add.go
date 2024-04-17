package cmd

import (
	"io"
	"strings"
	"time"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-registry/pkg/image/containerdregistry"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	internalaction "github.com/operator-framework/kubectl-operator/internal/pkg/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

func newCatalogAddCmd(cfg *action.Configuration) *cobra.Command {
	a := internalaction.NewCatalogAdd(cfg)
	a.Logf = log.Printf

	var extractContentStr string

	cmd := &cobra.Command{
		Use:   "add <name> <index_image>",
		Short: "Add an operator catalog",
		Args:  cobra.ExactArgs(2),
		PreRun: func(cmd *cobra.Command, args []string) {
			regLogger := logrus.New()
			regLogger.SetOutput(io.Discard)
			a.RegistryOptions = []containerdregistry.RegistryOption{
				containerdregistry.WithLog(logrus.NewEntry(regLogger)),
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			a.CatalogSourceName = args[0]
			a.IndexImage = args[1]

			if extractContentStr != "" {
				extractContentSettings := strings.Split(extractContentStr, ",")
				extractContentMap := map[string]string{}
				for _, setting := range extractContentSettings {
					key, value, ok := strings.Cut(setting, "=")
					if !ok {
						log.Fatalf("invalid extract content setting %q: found key without value", setting)
					}
					extractContentMap[key] = value
				}
				if catalogDir, ok := extractContentMap["catalog"]; !ok {
					log.Fatal("invalid extract content: catalog not found")
				} else if cacheDir, ok := extractContentMap["cache"]; !ok {
					log.Fatal("invalid extract content: cache not found")
				} else {
					a.ExtractContent = &v1alpha1.ExtractContentConfig{
						CatalogDir: catalogDir,
						CacheDir:   cacheDir,
					}
				}
			}

			cs, err := a.Run(cmd.Context())
			if err != nil {
				log.Fatalf("failed to add catalog: %v", err)
			}
			log.Printf("created catalogsource %q\n", cs.Name)
		},
	}
	bindCatalogAddFlags(cmd.Flags(), a)
	cmd.Flags().StringVar(&extractContentStr, "extract-content", "", "Use OLM-provided catalog server with provided paths (e.g. --extract-content=catalog=/configs,cache=/tmp/cache)")

	return cmd
}

func bindCatalogAddFlags(fs *pflag.FlagSet, a *internalaction.CatalogAdd) {
	fs.StringVarP(&a.DisplayName, "display-name", "d", "", "display name of the index")
	fs.StringVarP(&a.Publisher, "publisher", "p", "", "publisher of the index")
	fs.StringVar(&a.SecurityContextConfig, "security-context-config", "restricted", "security context config to use to run the catalog")
	fs.DurationVar(&a.CleanupTimeout, "cleanup-timeout", time.Minute, "the amount of time to wait before cancelling cleanup")

	fs.StringArrayVarP(&a.InjectBundles, "inject-bundles", "b", nil, "inject extra bundles into the index at runtime")
	fs.StringVarP(&a.InjectBundleMode, "inject-bundle-mode", "m", "", "mode to use to inject bundles")
	_ = fs.MarkHidden("inject-bundle-mode")
}
