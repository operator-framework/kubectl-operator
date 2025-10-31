package olmv1

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	v1action "github.com/operator-framework/kubectl-operator/internal/pkg/v1/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type extentionInstallOptions struct {
	CatalogSelector string
}

func NewExtensionInstallCmd(cfg *action.Configuration) *cobra.Command {
	i := v1action.NewExtensionInstall(cfg)
	i.Logf = log.Printf
	var extentionInstallOpts extentionInstallOptions
	var err error

	cmd := &cobra.Command{
		Use:   "extension <extension_name>",
		Short: "Install an extension",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			i.ExtensionName = args[0]

			if len(extentionInstallOpts.CatalogSelector) != 0 {
				i.CatalogSelector, err = metav1.ParseToLabelSelector(extentionInstallOpts.CatalogSelector)
				if err != nil {
					log.Fatalf("unable to parse selector %s: %v", extentionInstallOpts.CatalogSelector, err)
				}
			}
			switch i.UpgradeConstraintPolicy {
			case "CatalogProvided", "SelfCertified", "":
			default:
				log.Fatalf("unrecognized Upgrade Constraint Policy %s, must be one of: (CatalogProvided|SelfCertified)", i.UpgradeConstraintPolicy)
			}
			switch i.CRDUpgradeSafetyEnforcement {
			case "Strict", "None", "":
			default:
				log.Fatalf("unrecognized CRD Upgrade Safety Enforcement Policy %s, must be one of: (Strict|None)", i.CRDUpgradeSafetyEnforcement)
			}
			if len(i.DryRun) > 0 && i.DryRun != v1action.DryRunAll {
				log.Fatalf("invalid value for `--dry-run` %s, must be one of (%s)\n", i.DryRun, v1action.DryRunAll)
			}
			extObj, err := i.Run(cmd.Context())
			if err != nil {
				log.Fatalf("failed to install extension: %v", err)
			}
			log.Printf("extension %q created", i.ExtensionName)

			if err != nil {
				log.Fatalf("failed to install extension %q: %v\n", i.ExtensionName, err)
			}
			if len(i.DryRun) == 0 {
				log.Printf("extension %q created\n", i.ExtensionName)
				return
			}
			if len(i.Output) == 0 {
				log.Printf("extension %q created (dry run)\n", i.ExtensionName)
				return
			}

			extObj.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{Group: olmv1.GroupVersion.Group,
				Version: olmv1.GroupVersion.Version, Kind: olmv1.ClusterExtensionKind})
			printFormattedExtensions(i.Output, *extObj)

		},
	}
	bindExtensionInstallFlags(cmd.Flags(), i, &extentionInstallOpts)

	return cmd
}

func bindExtensionInstallFlags(fs *pflag.FlagSet, i *v1action.ExtensionInstall, o *extentionInstallOptions) {
	fs.StringVarP(&i.Namespace.Name, "namespace", "n", "", "namespace to install the operator in") //infer?
	fs.StringVarP(&i.PackageName, "package-name", "p", "", "package name of the operator to install. Required.")
	fs.StringSliceVarP(&i.Channels, "channels", "c", []string{}, "channels which would be to used for getting updates e.g --channels \"stable,dev-preview,preview\"")
	fs.StringVarP(&i.Version, "version", "v", "", "version (or version range) from which to resolve bundles")
	fs.StringVarP(&i.ServiceAccount, "service-account", "s", "default", "service account name to use for the extension installation")
	fs.DurationVarP(&i.CleanupTimeout, "cleanup-timeout", "d", time.Minute, "the amount of time to wait before cancelling cleanup after a failed creation attempt")
	fs.StringToStringVar(&i.Labels, "labels", map[string]string{}, "labels to add to the new extension")
	fs.StringVar(&i.DryRun, "dry-run", "", "display the object that would be sent on a request without applying it. One of: (All)")
	fs.StringVarP(&i.Output, "output", "o", "", "output format for dry-run manifests. One of: (json, yaml)")
	fs.StringVar(&o.CatalogSelector, "catalog-selector", "", "selector (label query) to filter catalogs to search for the package, "+
		"supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 "+
		"in (value3)). Matching objects must satisfy all of the specified label constraints.")
	fs.StringVar(&i.UpgradeConstraintPolicy, "upgrade-constraint-policy", "CatalogProvided", "controls whether the upgrade path(s) defined in the catalog are enforced."+
		" One of CatalogProvided, SelfCertified), Default: CatalogProvided")
	fs.StringVar(&i.CRDUpgradeSafetyEnforcement, "crd-upgrade-safety-enforcement", "Strict", "policy for preflight CRD Upgrade safety checks. One of: (Strict, None), default: Strict")
}
