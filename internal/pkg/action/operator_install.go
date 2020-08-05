package action

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	v1 "github.com/operator-framework/api/pkg/operators/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	operatorsv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/apis/operators/v1"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/joelanford/kubectl-operator/internal/pkg/log"
	"github.com/joelanford/kubectl-operator/internal/pkg/operator"
	"github.com/joelanford/kubectl-operator/internal/pkg/subscription"
)

type OperatorInstall struct {
	config *Configuration

	Package             string
	Channel             string
	Version             string
	Approval            subscription.ApprovalValue
	WatchNamespaces     []string
	InstallMode         operator.InstallMode
	InstallTimeout      time.Duration
	CleanupTimeout      time.Duration
	CreateOperatorGroup bool
}

func NewOperatorInstall(cfg *Configuration) *OperatorInstall {
	return &OperatorInstall{
		config: cfg,
	}
}

func (i *OperatorInstall) BindFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&i.Channel, "channel", "c", "", "subscription channel")
	fs.VarP(&i.Approval, "approval", "a", fmt.Sprintf("approval (%s or %s)", v1alpha1.ApprovalManual, v1alpha1.ApprovalAutomatic))
	fs.StringVarP(&i.Version, "version", "v", "", "install specific version for operator (default latest)")
	fs.StringSliceVarP(&i.WatchNamespaces, "watch", "w", []string{}, "namespaces to watch")
	fs.DurationVarP(&i.InstallTimeout, "timeout", "t", time.Minute, "the amount of time to wait before cancelling the install")
	fs.DurationVar(&i.CleanupTimeout, "cleanup-timeout", time.Minute, "the amount to time to wait before cancelling cleanup")
	fs.BoolVarP(&i.CreateOperatorGroup, "create-operator-group", "C", false, "create operator group if necessary")

	fs.VarP(&i.InstallMode, "install-mode", "i", "install mode")
	err := fs.MarkHidden("install-mode")
	if err != nil {
		log.Print(`requested flag "install-mode" missing`)
	}
}

func (i *OperatorInstall) Run(ctx context.Context) (*v1alpha1.ClusterServiceVersion, error) {
	if len(i.WatchNamespaces) > 0 && !i.InstallMode.IsEmpty() {
		return nil, fmt.Errorf("WatchNamespaces and InstallMode options are mutually exclusive")
	}
	if i.InstallMode.IsEmpty() {
		i.configureInstallModeFromWatch()
	}

	pm, err := i.getPackageManifest(ctx)
	if err != nil {
		return nil, fmt.Errorf("get package manifest: %v", err)
	}

	pc, err := i.getPackageChannel(pm)
	if err != nil {
		return nil, fmt.Errorf("get package channel: %v", err)
	}

	if _, err := i.ensureOperatorGroup(ctx, pm, pc); err != nil {
		return nil, err
	}

	sub, err := i.createSubscription(ctx, pm, pc)
	if err != nil {
		return nil, err
	}
	log.Printf("subscription %q created", sub.Name)

	ip, err := i.getInstallPlan(ctx, sub)
	if err != nil {
		return nil, err
	}

	// We need to approve the initial install plan
	if i.Approval.Approval == v1alpha1.ApprovalManual {
		if err := i.approveInstallPlan(ctx, ip); err != nil {
			return nil, err
		}
	}

	csv, err := i.getCSV(ctx, ip)
	if err != nil {
		return nil, fmt.Errorf("get clusterserviceversion: %v", err)
	}
	return csv, nil
}

func (i *OperatorInstall) configureInstallModeFromWatch() {
	i.InstallMode.TargetNamespaces = i.WatchNamespaces
	switch len(i.InstallMode.TargetNamespaces) {
	case 0:
		i.InstallMode.InstallModeType = v1alpha1.InstallModeTypeAllNamespaces
	case 1:
		if i.InstallMode.TargetNamespaces[0] == i.config.Namespace {
			i.InstallMode.InstallModeType = v1alpha1.InstallModeTypeOwnNamespace
		} else {
			i.InstallMode.InstallModeType = v1alpha1.InstallModeTypeSingleNamespace
		}
	default:
		i.InstallMode.InstallModeType = v1alpha1.InstallModeTypeMultiNamespace
	}
}

func (i *OperatorInstall) getPackageManifest(ctx context.Context) (*operatorsv1.PackageManifest, error) {
	pm := &operatorsv1.PackageManifest{}
	key := types.NamespacedName{
		Namespace: i.config.Namespace,
		Name:      i.Package,
	}
	if err := i.config.Client.Get(ctx, key, pm); err != nil {
		return nil, err
	}
	return pm, nil
}

func (i *OperatorInstall) getPackageChannel(pm *operatorsv1.PackageManifest) (*operatorsv1.PackageChannel, error) {
	if i.Channel == "" {
		i.Channel = pm.Status.DefaultChannel
	}
	var packageChannel *operatorsv1.PackageChannel
	for idx, ch := range pm.Status.Channels {
		if ch.Name == i.Channel {
			packageChannel = &pm.Status.Channels[idx]
		}
	}
	if packageChannel == nil {
		return nil, fmt.Errorf("channel %q does not exist for package %q", i.Channel, i.Package)
	}
	return packageChannel, nil
}

func (i *OperatorInstall) ensureOperatorGroup(ctx context.Context, pm *operatorsv1.PackageManifest, pc *operatorsv1.PackageChannel) (*v1.OperatorGroup, error) {
	og, err := i.getOperatorGroup(ctx)
	if err != nil {
		return nil, err
	}

	supported := getSupportedInstallModes(pc.CurrentCSVDesc.InstallModes)
	if supported.Len() == 0 {
		return nil, fmt.Errorf("operator %q is not installable: no supported install modes", pm.Name)
	}

	if !i.InstallMode.IsEmpty() {
		if i.InstallMode.InstallModeType == v1alpha1.InstallModeTypeSingleNamespace || i.InstallMode.InstallModeType == v1alpha1.InstallModeTypeMultiNamespace {
			targetNsSet := sets.NewString(i.InstallMode.TargetNamespaces...)
			if !supported.Has(string(v1alpha1.InstallModeTypeOwnNamespace)) && targetNsSet.Has(i.config.Namespace) {
				return nil, fmt.Errorf("cannot watch namespace %q: operator %q does not support install mode %q", i.config.Namespace, pm.Name, v1alpha1.InstallModeTypeOwnNamespace)
			}
		}
		if i.InstallMode.InstallModeType == v1alpha1.InstallModeTypeSingleNamespace && i.InstallMode.TargetNamespaces[0] == i.config.Namespace {
			return nil, fmt.Errorf("use install mode %q to watch operator's namespace %q", v1alpha1.InstallModeTypeOwnNamespace, i.config.Namespace)
		}

		supported = supported.Intersection(sets.NewString(string(i.InstallMode.InstallModeType)))
		if supported.Len() == 0 {
			return nil, fmt.Errorf("operator %q does not support install mode %q", pm.Name, i.InstallMode.InstallModeType)
		}
	}

	if og == nil {
		if i.CreateOperatorGroup {
			targetNamespaces, err := i.getTargetNamespaces(supported)
			if err != nil {
				return nil, err
			}
			if og, err = i.createOperatorGroup(ctx, targetNamespaces); err != nil {
				return nil, fmt.Errorf("create operator group: %v", err)
			}
			log.Printf("operatorgroup %q created", og.Name)
		} else {
			return nil, fmt.Errorf("namespace %q has no existing operator group; use --create-operator-group to create one automatically", i.config.Namespace)
		}
	} else if err := i.validateOperatorGroup(*og, supported); err != nil {
		return nil, err
	}
	return og, nil
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

func getSupportedInstallModes(csvInstallModes []v1alpha1.InstallMode) sets.String {
	supported := sets.NewString()
	for _, im := range csvInstallModes {
		if im.Supported {
			supported.Insert(string(im.Type))
		}
	}
	return supported
}

func (i *OperatorInstall) getTargetNamespaces(supported sets.String) ([]string, error) {
	switch {
	case supported.Has(string(v1alpha1.InstallModeTypeAllNamespaces)):
		return nil, nil
	case supported.Has(string(v1alpha1.InstallModeTypeOwnNamespace)):
		return []string{i.config.Namespace}, nil
	case supported.Has(string(v1alpha1.InstallModeTypeSingleNamespace)):
		if len(i.InstallMode.TargetNamespaces) != 1 {
			return nil, fmt.Errorf("install mode %q requires explicit target namespace", v1alpha1.InstallModeTypeSingleNamespace)
		}
		return i.InstallMode.TargetNamespaces, nil
	case supported.Has(string(v1alpha1.InstallModeTypeMultiNamespace)):
		if len(i.InstallMode.TargetNamespaces) == 0 {
			return nil, fmt.Errorf("install mode %q requires explicit target namespaces", v1alpha1.InstallModeTypeMultiNamespace)
		}
		return i.InstallMode.TargetNamespaces, nil
	default:
		return nil, fmt.Errorf("no supported install modes")
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

func (i *OperatorInstall) validateOperatorGroup(og v1.OperatorGroup, supported sets.String) error {
	ogTargetNs := sets.NewString(og.Spec.TargetNamespaces...)
	imTargetNs := sets.NewString(i.InstallMode.TargetNamespaces...)
	ownNamespaceNs := sets.NewString(i.config.Namespace)

	if supported.Has(string(v1alpha1.InstallModeTypeAllNamespaces)) && len(og.Spec.TargetNamespaces) == 0 ||
		supported.Has(string(v1alpha1.InstallModeTypeOwnNamespace)) && ogTargetNs.Equal(ownNamespaceNs) ||
		supported.Has(string(v1alpha1.InstallModeTypeSingleNamespace)) && ogTargetNs.Equal(imTargetNs) ||
		supported.Has(string(v1alpha1.InstallModeTypeMultiNamespace)) && ogTargetNs.Equal(imTargetNs) {
		return nil
	}

	switch i.InstallMode.InstallModeType {
	case v1alpha1.InstallModeTypeAllNamespaces, v1alpha1.InstallModeTypeOwnNamespace,
		v1alpha1.InstallModeTypeSingleNamespace, v1alpha1.InstallModeTypeMultiNamespace:
		return fmt.Errorf("existing operatorgroup %q is not compatible with install mode %q", og.Name, i.InstallMode)
	case "":
		return fmt.Errorf("existing operatorgroup %q is not compatible with any supported package install modes", og.Name)
	}
	panic(fmt.Sprintf("unknown install mode %q", i.InstallMode.InstallModeType))
}

func (i *OperatorInstall) createSubscription(ctx context.Context, pm *operatorsv1.PackageManifest, pc *operatorsv1.PackageChannel) (*v1alpha1.Subscription, error) {
	opts := []subscription.Option{
		subscription.InstallPlanApproval(i.Approval.Approval),
	}

	if i.Version != "" {
		guessedStartingCSV := guessStartingCSV(pc.CurrentCSV, i.Version)
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

func guessStartingCSV(csvNameExample, desiredVersion string) string {
	csvBaseName, vChar, _ := parseCSVName(csvNameExample)
	version := strings.TrimPrefix(desiredVersion, "v")
	return fmt.Sprintf("%s.%s%s", csvBaseName, vChar, version)
}

const (
	operatorNameRegexp = `[a-z0-9]([-a-z0-9]*[a-z0-9])?`
	semverRegexp       = `(?P<major>0|[1-9]\d*)\.(?P<minor>0|[1-9]\d*)\.(?P<patch>0|[1-9]\d*)(?:-(?P<prerelease>(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+(?P<buildmetadata>[0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?` //nolint:lll
)

var csvNameRegexp = regexp.MustCompile(`^(` + operatorNameRegexp + `).(v?)(` + semverRegexp + `)$`)

func parseCSVName(csvName string) (string, string, string) {
	matches := csvNameRegexp.FindAllStringSubmatch(csvName, -1)
	return matches[0][1], matches[0][3], matches[0][4]
}

func (i *OperatorInstall) getInstallPlan(ctx context.Context, sub *v1alpha1.Subscription) (*v1alpha1.InstallPlan, error) {
	subKey := types.NamespacedName{
		Namespace: sub.GetNamespace(),
		Name:      sub.GetName(),
	}
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

func (i *OperatorInstall) approveInstallPlan(ctx context.Context, ip *v1alpha1.InstallPlan) error {
	ip.Spec.Approved = true
	if err := i.config.Client.Update(ctx, ip); err != nil {
		return fmt.Errorf("approve install plan: %v", err)
	}
	return nil
}

func (i *OperatorInstall) getCSV(ctx context.Context, ip *v1alpha1.InstallPlan) (*v1alpha1.ClusterServiceVersion, error) {
	ipKey := types.NamespacedName{
		Namespace: ip.GetNamespace(),
		Name:      ip.GetName(),
	}
	if err := wait.PollImmediateUntil(time.Millisecond*250, func() (bool, error) {
		if err := i.config.Client.Get(ctx, ipKey, ip); err != nil {
			return false, err
		}
		if ip.Status.Phase == v1alpha1.InstallPlanPhaseComplete {
			return true, nil
		}
		return false, nil
	}, ctx.Done()); err != nil {
		return nil, fmt.Errorf("waiting for operator installation to complete: %v", err)
	}

	csvKey := types.NamespacedName{
		Namespace: i.config.Namespace,
	}
	for _, s := range ip.Status.Plan {
		if s.Resource.Kind == "ClusterServiceVersion" {
			csvKey.Name = s.Resource.Name
		}
	}
	if csvKey.Name == "" {
		return nil, fmt.Errorf("could not find installed CSV in install plan")
	}
	csv := &v1alpha1.ClusterServiceVersion{}
	if err := i.config.Client.Get(ctx, csvKey, csv); err != nil {
		return nil, fmt.Errorf("get clusterserviceversion: %v", err)
	}
	return csv, nil
}
