package action

import (
	"context"
	"flag"
	"fmt"
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

type InstallOperator struct {
	config *Configuration

	Package             string
	Channel             string
	Approve             string
	InstallMode         operator.InstallMode
	InstallTimeout      time.Duration
	CleanupTimeout      time.Duration
	CreateOperatorGroup bool
}

func NewInstallOperator(cfg *Configuration) *InstallOperator {
	return &InstallOperator{
		config: cfg,
	}
}

func (i *InstallOperator) BindFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&i.Channel, "channel", "c", "", "subscription channel")
	fs.StringVarP(&i.Approve, "approval", "a", "", "approval (Automatic or Manual)")
	fs.DurationVarP(&i.InstallTimeout, "timeout", "t", time.Minute, "the amount of time to wait before cancelling the install")
	fs.DurationVar(&i.CleanupTimeout, "cleanup-timeout", time.Minute, "the amount to time to wait before cancelling cleanup")
	fs.BoolVar(&i.CreateOperatorGroup, "create-operator-group", false, "create operator group if necessary")
	imVal := pflag.PFlagFromGoFlag(&flag.Flag{Value: &i.InstallMode}).Value
	fs.VarP(imVal, "install-mode", "i", "install mode")
}

func (i *InstallOperator) Run(ctx context.Context) (*v1alpha1.ClusterServiceVersion, error) {
	og, err := i.getOperatorGroup(ctx)
	if err != nil {
		return nil, err
	}

	pm, err := i.getPackageManifest(ctx)
	if err != nil {
		return nil, fmt.Errorf("get package manifest: %v", err)
	}

	pc, err := i.getPackageChannel(pm)
	if err != nil {
		return nil, fmt.Errorf("get package channel: %v", err)
	}
	supported := getSupportedInstallModes(pc.CurrentCSVDesc.InstallModes)
	if !i.InstallMode.IsEmpty() {
		supported = supported.Intersection(sets.NewString(string(i.InstallMode.InstallModeType)))
		if supported.Len() == 0 {
			return nil, fmt.Errorf("operator %q does not support install mode %q", pm.Name, i.InstallMode.InstallModeType)
		}
	}

	if og == nil {
		if i.CreateOperatorGroup {
			if og, err = i.createOperatorGroup(ctx, supported); err != nil {
				return nil, fmt.Errorf("create operator group: %v", err)
			}
			log.Printf("operatorgroup %q created", og.Name)
		} else {
			return nil, fmt.Errorf("namespace %q has no existing operator group", i.config.Namespace)
		}
	} else if err := i.validateOperatorGroup(*og, supported); err != nil {
		return nil, err
	}

	opts := []subscription.Option{}
	if i.Approve != "" {
		opts = append(opts, subscription.InstallPlanApproval(v1alpha1.Approval(i.Approve)))
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
	log.Printf("subscription %q created", sub.Name)

	if err := wait.PollImmediateUntil(time.Millisecond*250, func() (bool, error) {
		if err := i.config.Client.Get(ctx, subKey, sub); err != nil {
			return false, err
		}
		if sub.Status.State == v1alpha1.SubscriptionStateAtLatest {
			return true, nil
		}
		return false, nil
	}, ctx.Done()); err != nil {
		return nil, fmt.Errorf("waiting for state \"AtLatestKnown\": %v", err)
	}

	csvKey := types.NamespacedName{
		Namespace: i.config.Namespace,
		Name:      sub.Status.InstalledCSV,
	}
	csv := &v1alpha1.ClusterServiceVersion{}
	if err := i.config.Client.Get(ctx, csvKey, csv); err != nil {
		return nil, fmt.Errorf("get clusterserviceversion: %v", err)
	}
	return csv, nil
}

func (i InstallOperator) getOperatorGroup(ctx context.Context) (*v1.OperatorGroup, error) {
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

func (i *InstallOperator) getPackageManifest(ctx context.Context) (*operatorsv1.PackageManifest, error) {
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

func (i *InstallOperator) createOperatorGroup(ctx context.Context, supported sets.String) (*v1.OperatorGroup, error) {
	og := &v1.OperatorGroup{}
	og.SetName(i.config.Namespace)
	og.SetNamespace(i.config.Namespace)

	switch {
	case supported.HasAny(
		string(v1alpha1.InstallModeTypeAllNamespaces),
		string(v1alpha1.InstallModeTypeSingleNamespace),
		string(v1alpha1.InstallModeTypeMultiNamespace)):
		og.Spec.TargetNamespaces = i.InstallMode.TargetNamespaces
	case supported.Has(string(v1alpha1.InstallModeTypeOwnNamespace)):
		og.Spec.TargetNamespaces = []string{i.config.Namespace}
	default:
		return nil, fmt.Errorf("no supported install modes")
	}
	if err := i.config.Client.Create(ctx, og); err != nil {
		return nil, err
	}
	return og, nil
}

func (i *InstallOperator) validateOperatorGroup(og v1.OperatorGroup, supported sets.String) error {
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

func getSupportedInstallModes(csvInstallModes []v1alpha1.InstallMode) sets.String {
	supported := sets.NewString()
	for _, im := range csvInstallModes {
		if im.Supported {
			supported.Insert(string(im.Type))
		}
	}
	return supported
}

func (i *InstallOperator) getPackageChannel(pm *operatorsv1.PackageManifest) (*operatorsv1.PackageChannel, error) {
	if i.Channel == "" {
		i.Channel = pm.Status.DefaultChannel
	}
	var packageChannel *operatorsv1.PackageChannel
	for _, ch := range pm.Status.Channels {
		if ch.Name == i.Channel {
			packageChannel = &ch
		}
	}
	if packageChannel == nil {
		return nil, fmt.Errorf("channel %q does not exist for package %q", i.Channel, i.Package)
	}
	return packageChannel, nil
}

func (i *InstallOperator) cleanup(ctx context.Context, sub *v1alpha1.Subscription) {
	if err := i.config.Client.Delete(ctx, sub); err != nil {
		log.Printf("delete subscription %q: %v", sub.Name, err)
	}
}
