package olmv1

import (
	"fmt"

	"github.com/blang/semver/v4"
	"github.com/containerd/containerd/reference"
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	v1action "github.com/operator-framework/kubectl-operator/internal/pkg/v1/action"
)

type getOptions struct {
	Output   string
	Selector string
}

func bindGetFlags(fs *pflag.FlagSet, o *getOptions) {
	fs.StringVarP(&o.Output, "output", "o", "", "output format. One of: (json, yaml)")
	fs.StringVarP(&o.Selector, "selector", "l", "", "selector (label query) to filter on, "+
		"supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 "+
		"in (value3)). Matching objects must satisfy all of the specified label constraints.")

}

type dryRunOptions struct {
	DryRun string
	Output string
}

func bindDryRunFlags(fs *pflag.FlagSet, o *dryRunOptions) {
	fs.StringVar(&o.DryRun, "dry-run", "", "display the object that would be sent on a request without applying it. One of: (All)")
	fs.StringVarP(&o.Output, "output", "o", "", "output format for dry-run manifests. One of: (json, yaml)")
}

func (o *dryRunOptions) validate() error {
	var errs []error
	if len(o.DryRun) > 0 && o.DryRun != v1action.DryRunAll {
		errs = append(errs, fmt.Errorf("invalid value for `--dry-run` %s, must be one of (%s)", o.DryRun, v1action.DryRunAll))
	}
	switch o.Output {
	case "json", "yaml", "":
	default:
		errs = append(errs, fmt.Errorf("unrecognized output format %s: must be one of (json, yaml)", o.Output))
	}
	return errors.NewAggregate(errs)
}

type mutableExtensionOptions struct {
	Channels                    []string
	Version                     string
	Labels                      map[string]string
	UpgradeConstraintPolicy     string
	CRDUpgradeSafetyEnforcement string
	CatalogSelector             string
	ParsedSelector              *metav1.LabelSelector
}

func bindMutableExtensionFlags(fs *pflag.FlagSet, o *mutableExtensionOptions) {
	fs.StringSliceVarP(&o.Channels, "channels", "c", []string{}, "channels to be used for getting updates. If omitted, extension versions in all channels will be "+
		"considered for upgrades. When used with '--version', only package versions meeting both constraints will be considered.")
	fs.StringVarP(&o.Version, "version", "v", "", "version (or version range) in semver format to limit the allowable package versions to. If used with '--channel', "+
		"only package versions meeting both constraints will be considered.")
	fs.StringToStringVar(&o.Labels, "labels", map[string]string{}, "labels to add to the extension. Set a label's value as empty to remove that label")
	fs.StringVar(&o.CRDUpgradeSafetyEnforcement, "crd-upgrade-safety-enforcement", "", fmt.Sprintf("policy for preflight CRD Upgrade safety checks. One of: %v, (default %s)",
		[]string{string(olmv1.CRDUpgradeSafetyEnforcementStrict), string(olmv1.CRDUpgradeSafetyEnforcementNone)}, olmv1.CRDUpgradeSafetyEnforcementStrict))
	fs.StringVar(&o.UpgradeConstraintPolicy, "upgrade-constraint-policy", "", "controls whether the package upgrade path(s) defined in the catalog are enforced."+
		fmt.Sprintf(" One of %v, (default %s)", []string{string(olmv1.UpgradeConstraintPolicyCatalogProvided), string(olmv1.UpgradeConstraintPolicySelfCertified)},
			olmv1.UpgradeConstraintPolicyCatalogProvided))
	fs.StringVarP(&o.CatalogSelector, "catalog-selector", "l", "", "selector (label query) to filter catalogs to search for the package, "+
		"supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 "+
		"in (value3)). Matching objects must satisfy all of the specified label constraints.")
}

func (o mutableExtensionOptions) validate() error {
	var errs []error
	if len(o.Version) > 0 {
		if _, err := semver.ParseRange(o.Version); err != nil {
			errs = append(errs, fmt.Errorf("invalid `--version` %s: %v", o.Version, err))
		}
	}
	switch o.CRDUpgradeSafetyEnforcement {
	case string(olmv1.CRDUpgradeSafetyEnforcementStrict), string(olmv1.CRDUpgradeSafetyEnforcementNone), "":
	default:
		errs = append(errs, fmt.Errorf("invalid `--crd-upgrade-safety-enforcement` %s: must be one of: %v", o.CRDUpgradeSafetyEnforcement,
			[]string{string(olmv1.CRDUpgradeSafetyEnforcementStrict), string(olmv1.CRDUpgradeSafetyEnforcementNone)}))
	}
	switch o.UpgradeConstraintPolicy {
	case string(olmv1.UpgradeConstraintPolicyCatalogProvided), string(olmv1.UpgradeConstraintPolicySelfCertified), "":
	default:
		errs = append(errs, fmt.Errorf("invalid `--upgrade-constraint-policy` %s: must be one of: %v", o.UpgradeConstraintPolicy,
			[]string{string(olmv1.UpgradeConstraintPolicyCatalogProvided), string(olmv1.UpgradeConstraintPolicySelfCertified)}))
	}
	if len(o.CatalogSelector) > 0 {
		var err error
		o.ParsedSelector, err = metav1.ParseToLabelSelector(o.CatalogSelector)
		if err != nil {
			errs = append(errs, fmt.Errorf("invalid `--labels` value %s: %v", o.CatalogSelector, err))
		}
	}
	return errors.NewAggregate(errs)
}

type mutableCatalogOptions struct {
	Priority            int32
	AvailabilityMode    string
	PollIntervalMinutes int
	Labels              map[string]string
	Image               string
}

func bindMutableCatalogFlags(fs *pflag.FlagSet, o *mutableCatalogOptions) {
	fs.Int32Var(&o.Priority, "priority", 0, "relative priority of the catalog among all on-cluster catalogs for installing or updating packages."+
		" A higher number equals greater priority; negative values indicate less priority than the default.")
	fs.StringVar(&o.AvailabilityMode, "available", "", "determines whether a catalog should be active and serving data. Setting the flag to false "+
		"means the catalog will not serve its contents. Set to true by default for new catalogs.")
	fs.IntVar(&o.PollIntervalMinutes, "source-poll-interval-minutes", 0, "the interval in minutes to poll the catalog's source image for new content."+
		" Only valid for tag based source image references. Set to 0 or -1 to disable polling.")
	fs.StringToStringVar(&o.Labels, "labels", map[string]string{}, "labels to add to the catalog. Set a label's value as empty to remove it.")
}

func (o mutableCatalogOptions) validate() error {
	var errs []error
	switch o.AvailabilityMode {
	case "":
	case "true":
		o.AvailabilityMode = string(olmv1.AvailabilityModeAvailable)
	case "false":
		o.AvailabilityMode = string(olmv1.AvailabilityModeUnavailable)
	default:
		errs = append(errs, fmt.Errorf("invalid `--labels` value %s: must be one of: [true, false]", o.AvailabilityMode))
	}
	if o.PollIntervalMinutes > 0 && len(o.Image) > 0 {
		ref, err := reference.Parse(o.Image)
		if err != nil {
			errs = append(errs, fmt.Errorf("invalid catalog source image %s: %v", o.Image, err))
		} else if len(ref.Digest()) != 0 {
			errs = append(errs, fmt.Errorf("cannot specify a non-zero --source-poll-interval-minutes for a digest based catalog image %s", o.Image))
		}
	}
	return errors.NewAggregate(errs)
}
