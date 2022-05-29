package action

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	v1 "github.com/operator-framework/api/pkg/operators/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	operatorsv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/apis/operators/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/operator-framework/kubectl-operator/internal/pkg/operator"
	"github.com/operator-framework/kubectl-operator/internal/pkg/subscription"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

type OperatorInstall struct {
	config *action.Configuration

	Package                  string
	Channel                  string
	Version                  string
	Approval                 subscription.ApprovalValue
	WatchNamespaces          []string
	CleanupTimeout           time.Duration
	CreateOperatorGroup      bool
	PermitAlternateNamespace bool

	Logf func(string, ...interface{})
}

func NewOperatorInstall(cfg *action.Configuration) *OperatorInstall {
	return &OperatorInstall{
		config: cfg,
		Logf:   func(string, ...interface{}) {},
	}
}

var ErrNoOperatorGroup = errors.New("operator group not found")

type ErrIncorrectNamespace struct {
	Requested string
	Suggested string
}

func (e ErrIncorrectNamespace) Error() string {
	return fmt.Sprintf("requested install namespace is %q, but operator's suggested namespace is %q", e.Requested, e.Suggested)
}

func (i *OperatorInstall) Run(ctx context.Context) (*v1alpha1.ClusterServiceVersion, error) {
	pm, err := i.getPackageManifest(ctx)
	if err != nil {
		return nil, fmt.Errorf("get package manifest: %v", err)
	}

	pc, err := pm.GetChannel(i.Channel)
	if err != nil {
		return nil, fmt.Errorf("get package channel: %v", err)
	}

	if err := i.ensureNamespace(ctx, pc); err != nil {
		return nil, err
	}

	if _, err := i.ensureOperatorGroup(ctx, pm, pc); err != nil {
		return nil, err
	}

	sub, err := i.createSubscription(ctx, pm, pc)
	if err != nil {
		return nil, err
	}
	i.Logf("subscription %q created", sub.Name)

	ip, err := i.getInstallPlan(ctx, sub)
	if err != nil {
		return nil, err
	}

	// We need to approve the initial install plan
	if i.Approval.Approval == v1alpha1.ApprovalManual {
		if err := approveInstallPlan(ctx, i.config.Client, ip); err != nil {
			return nil, fmt.Errorf("approve install plan: %v", err)
		}
	}

	csv, err := getCSV(ctx, i.config.Client, ip)
	if err != nil {
		return nil, fmt.Errorf("get clusterserviceversion: %v", err)
	}
	return csv, nil
}

func (i *OperatorInstall) possibleInstallModes(watchNamespaces []string) sets.String {
	switch len(watchNamespaces) {
	case 0:
		return sets.NewString(
			string(v1alpha1.InstallModeTypeAllNamespaces),
			string(v1alpha1.InstallModeTypeOwnNamespace),
		)
	case 1:
		if watchNamespaces[0] == i.config.Namespace {
			return sets.NewString(string(v1alpha1.InstallModeTypeOwnNamespace))
		}
		return sets.NewString(string(v1alpha1.InstallModeTypeSingleNamespace))
	default:
		return sets.NewString(string(v1alpha1.InstallModeTypeMultiNamespace))
	}
}

func (i *OperatorInstall) getPackageManifest(ctx context.Context) (*operator.PackageManifest, error) {
	pm := &operatorsv1.PackageManifest{}
	key := types.NamespacedName{
		Namespace: i.config.Namespace,
		Name:      i.Package,
	}
	if err := i.config.Client.Get(ctx, key, pm); err != nil {
		return nil, err
	}
	return &operator.PackageManifest{PackageManifest: *pm}, nil
}

func (i *OperatorInstall) ensureNamespace(ctx context.Context, pc *operator.PackageChannel) error {
	suggestedNamespace := pc.CurrentCSVDesc.Annotations["operatorframework.io/suggested-namespace"]
	if !i.PermitAlternateNamespace && suggestedNamespace != "" && i.config.Namespace != suggestedNamespace {
		return ErrIncorrectNamespace{Suggested: suggestedNamespace, Requested: i.config.Namespace}
	}
	ns := corev1.Namespace{}
	if err := i.config.Client.Get(ctx, types.NamespacedName{Name: i.config.Namespace}, &ns); err != nil {
		return err
	}
	return nil
}

func (i *OperatorInstall) ensureOperatorGroup(ctx context.Context, pm *operator.PackageManifest, pc *operator.PackageChannel) (*v1.OperatorGroup, error) {
	og, err := i.getOperatorGroup(ctx)
	if err != nil {
		return nil, err
	}

	operatorInstallModes := pc.GetSupportedInstallModes()
	if operatorInstallModes.Len() == 0 {
		return nil, fmt.Errorf("operator %q is not installable: operator defined no supported install modes", pm.Name)
	}

	desired := i.possibleInstallModes(i.WatchNamespaces)

	supported := operatorInstallModes.Intersection(desired)
	if supported.Len() == 0 {
		return nil, fmt.Errorf("operator %q is not installable: install modes supported by operator (%q) not compatible with install modes supported by desired watches (%q)",
			pm.Name,
			strings.Join(operatorInstallModes.List(), ","),
			strings.Join(desired.List(), ","),
		)
	}

	if og != nil {
		if err := i.validateOperatorGroup(*og, operatorInstallModes, desired); err != nil {
			return nil, fmt.Errorf("operator %q not installable: %v", pm.Name, err)
		}
	} else {
		if i.CreateOperatorGroup {
			targetNamespaces := i.getTargetNamespaces(supported)
			if og, err = i.createOperatorGroup(ctx, targetNamespaces); err != nil {
				return nil, fmt.Errorf("create operator group: %v", err)
			}
			i.Logf("operatorgroup %q created", og.Name)
		} else {
			return nil, ErrNoOperatorGroup
		}
	}
	return og, nil
}

func (i OperatorInstall) validateOperatorGroup(og v1.OperatorGroup, operatorInstallModes, desired sets.String) error {
	ogSupported := i.possibleInstallModes(og.Status.Namespaces)

	if operatorInstallModes.Intersection(ogSupported).Len() == 0 {
		return fmt.Errorf("install modes supported by operator (%q) not compatible with install modes supported by existing operator group (%q)",
			strings.Join(operatorInstallModes.List(), ","),
			strings.Join(ogSupported.List(), ","),
		)
	}

	if desired.Intersection(ogSupported).Len() == 0 {
		return fmt.Errorf("install modes supported by desired watches (%q) not compatible with install modes supported by existing operator group (%q)",
			strings.Join(desired.List(), ","),
			strings.Join(ogSupported.List(), ","),
		)
	}
	supported := operatorInstallModes.Intersection(desired)
	if supported.Intersection(ogSupported).Len() == 0 {
		return fmt.Errorf("install modes supported by operator and desired watches (%q) not compatible with install modes supported by existing operator group (%q)",
			strings.Join(supported.List(), ","),
			strings.Join(ogSupported.List(), ","),
		)
	}
	return nil
}

func (i OperatorInstall) getOperatorGroup(ctx context.Context) (*v1.OperatorGroup, error) {
	ogs := &v1.OperatorGroupList{}
	err := i.config.Client.List(ctx, ogs, client.InNamespace(i.config.Namespace))
	if err != nil {
		return nil, fmt.Errorf("list operator groups: %v", err)
	}

	switch len(ogs.Items) {
	case 0:
		return nil, nil
	case 1:
		return &ogs.Items[0], nil
	default:
		return nil, fmt.Errorf("namespace %q has more than one operator group", i.config.Namespace)
	}
}

func (i *OperatorInstall) getTargetNamespaces(supported sets.String) []string {
	switch {
	case supported.Has(string(v1alpha1.InstallModeTypeAllNamespaces)):
		return nil
	case supported.Has(string(v1alpha1.InstallModeTypeOwnNamespace)):
		return []string{i.config.Namespace}
	default:
		return i.WatchNamespaces
	}
}

func (i *OperatorInstall) createOperatorGroup(ctx context.Context, targetNamespaces []string) (*v1.OperatorGroup, error) {
	og := &v1.OperatorGroup{}
	og.SetName(i.config.Namespace)
	og.SetNamespace(i.config.Namespace)
	og.Spec.TargetNamespaces = targetNamespaces

	if err := i.config.Client.Create(ctx, og); err != nil {
		return nil, err
	}
	return og, nil
}

func (i *OperatorInstall) createSubscription(ctx context.Context, pm *operator.PackageManifest, pc *operator.PackageChannel) (*v1alpha1.Subscription, error) {
	opts := []subscription.Option{
		subscription.InstallPlanApproval(i.Approval.Approval),
	}

	if i.Version != "" {
		// Use the CSV name of the channel head as a template to guess the CSV name based on
		// the desired version.
		guessedStartingCSV, err := guessStartingCSV(pc.CurrentCSV, i.Version)
		if err != nil {
			return nil, fmt.Errorf("could not guess startingCSV: %v", err)
		}
		opts = append(opts, subscription.StartingCSV(guessedStartingCSV))
	}

	subKey := types.NamespacedName{
		Namespace: i.config.Namespace,
		Name:      i.Package,
	}
	sourceKey := types.NamespacedName{
		Namespace: pm.Status.CatalogSourceNamespace,
		Name:      pm.Status.CatalogSource,
	}
	sub := subscription.Build(subKey, i.Channel, sourceKey, opts...)
	if err := i.config.Client.Create(ctx, sub); err != nil {
		return nil, fmt.Errorf("create subscription: %v", err)

	}
	return sub, nil
}

// guessStartingCSV finds the first semver version string in csvNameExample, and replaces all
// occurrences with desiredVersion, trimming any "v" prefix from desiredVersion prior to making the
// replacements. If csvNameExample does not contain a semver version string, guessStartingCSV
// returns an error.
func guessStartingCSV(csvNameExample, desiredVersion string) (string, error) {
	exampleVersion := semverRegexp.FindString(csvNameExample)
	if exampleVersion == "" {
		return "", fmt.Errorf("could not locate semver version in channel head CSV name %q", csvNameExample)
	}
	desiredVersion = strings.TrimPrefix(desiredVersion, "v")
	return strings.ReplaceAll(csvNameExample, exampleVersion, desiredVersion), nil
}

var semverRegexp = regexp.MustCompile(`(?P<major>0|[1-9]\d*)\.(?P<minor>0|[1-9]\d*)\.(?P<patch>0|[1-9]\d*)(?:-(?P<prerelease>(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+(?P<buildmetadata>[0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?`) //nolint:lll

func (i *OperatorInstall) getInstallPlan(ctx context.Context, sub *v1alpha1.Subscription) (*v1alpha1.InstallPlan, error) {
	subKey := objectKeyForObject(sub)
	if err := wait.PollImmediateUntil(time.Millisecond*250, func() (bool, error) {
		if err := i.config.Client.Get(ctx, subKey, sub); err != nil {
			return false, err
		}
		if sub.Status.InstallPlanRef != nil {
			return true, nil
		}
		return false, nil
	}, ctx.Done()); err != nil {
		return nil, fmt.Errorf("waiting for install plan to exist: %v", err)
	}

	ip := v1alpha1.InstallPlan{}
	ipKey := types.NamespacedName{
		Namespace: sub.Status.InstallPlanRef.Namespace,
		Name:      sub.Status.InstallPlanRef.Name,
	}
	if err := i.config.Client.Get(ctx, ipKey, &ip); err != nil {
		return nil, fmt.Errorf("get install plan: %v", err)
	}
	return &ip, nil
}
