package action

import (
	"context"
	"errors"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	"github.com/operator-framework/kubectl-operator/pkg/action"
)

type ExtensionInstall struct {
	config        *action.Configuration
	ExtensionName string

	Namespace                            NamespaceConfig
	PackageName                          string
	Channels                             []string
	Version                              string
	CatalogSelector                      *metav1.LabelSelector
	ServiceAccount                       string
	CleanupTimeout                       time.Duration
	UpgradeConstraintPolicy              string
	PreflightCRDUpgradeSafetyEnforcement string
	CRDUpgradeSafetyEnforcement          string
	Labels                               map[string]string

	DryRun string
	Output string
	Logf   func(string, ...interface{})
}
type NamespaceConfig struct {
	Name string
}

func NewExtensionInstall(cfg *action.Configuration) *ExtensionInstall {
	return &ExtensionInstall{
		config: cfg,
		Logf:   func(string, ...interface{}) {},
	}
}

func (i *ExtensionInstall) buildClusterExtension() olmv1.ClusterExtension {
	extension := olmv1.ClusterExtension{
		ObjectMeta: metav1.ObjectMeta{
			Name:   i.ExtensionName,
			Labels: i.Labels,
		},
		Spec: olmv1.ClusterExtensionSpec{
			Source: olmv1.SourceConfig{
				SourceType: olmv1.SourceTypeCatalog,
				Catalog: &olmv1.CatalogFilter{
					PackageName: i.PackageName,
					Version:     i.Version,
				},
			},
			Namespace: i.Namespace.Name,
			ServiceAccount: olmv1.ServiceAccountReference{
				Name: i.ServiceAccount,
			},
		},
	}
	if i.CatalogSelector != nil {
		extension.Spec.Source.Catalog.Selector = i.CatalogSelector
	}
	if len(i.UpgradeConstraintPolicy) > 0 {
		extension.Spec.Source.Catalog.UpgradeConstraintPolicy = olmv1.UpgradeConstraintPolicy(i.UpgradeConstraintPolicy)
	}
	if len(i.CRDUpgradeSafetyEnforcement) > 0 {
		extension.Spec.Install = &olmv1.ClusterExtensionInstallConfig{
			Preflight: &olmv1.PreflightConfig{
				CRDUpgradeSafety: &olmv1.CRDUpgradeSafetyPreflightConfig{
					Enforcement: olmv1.CRDUpgradeSafetyEnforcement(i.CRDUpgradeSafetyEnforcement),
				},
			},
		}
	}

	return extension
}

func (i *ExtensionInstall) Run(ctx context.Context) (*olmv1.ClusterExtension, error) {
	extension := i.buildClusterExtension()

	// Add Channels to extension
	if len(i.Channels) > 0 {
		extension.Spec.Source.Catalog.Channels = i.Channels
	}

	if i.DryRun == DryRunAll {
		if err := i.config.Client.Create(ctx, &extension, client.DryRunAll); err != nil {
			return nil, err
		}
		return &extension, nil
	}
	// Create the extension
	if err := i.config.Client.Create(ctx, &extension); err != nil {
		return nil, err
	}
	clusterExtension, err := i.waitForExtensionInstall(ctx)
	if err != nil {
		i.Logf("failed to install extension %s: %w; cleaning up extension", i.PackageName, err)
		cleanupCtx, cancelCleanup := context.WithTimeout(context.Background(), i.CleanupTimeout)
		defer cancelCleanup()
		cleanupErr := i.cleanup(cleanupCtx)
		return nil, errors.Join(err, cleanupErr)
	}
	return clusterExtension, nil
}

// waitForClusterExtensionInstalled waits for the ClusterExtension to be installed
// and returns the ClusterExtension object
func (i *ExtensionInstall) waitForExtensionInstall(ctx context.Context) (*olmv1.ClusterExtension, error) {
	clusterExtension := &olmv1.ClusterExtension{
		ObjectMeta: metav1.ObjectMeta{
			Name: i.ExtensionName,
		},
	}
	errMsg := ""
	key := client.ObjectKeyFromObject(clusterExtension)
	if err := wait.PollUntilContextCancel(ctx, time.Millisecond*250, true, func(conditionCtx context.Context) (bool, error) {
		if err := i.config.Client.Get(conditionCtx, key, clusterExtension); err != nil {
			return false, err
		}
		progressingCondition := meta.FindStatusCondition(clusterExtension.Status.Conditions, olmv1.TypeProgressing)
		if progressingCondition != nil && progressingCondition.Reason != olmv1.ReasonSucceeded {
			errMsg = progressingCondition.Message
			return false, nil
		}
		if !meta.IsStatusConditionPresentAndEqual(clusterExtension.Status.Conditions, olmv1.TypeInstalled, metav1.ConditionTrue) {
			return false, nil
		}
		return true, nil
	}); err != nil {
		if errMsg == "" {
			errMsg = err.Error()
		}
		return nil, fmt.Errorf("cluster extension %q did not finish installing: %s", clusterExtension.Name, errMsg)
	}
	return clusterExtension, nil
}

func (i *ExtensionInstall) cleanup(ctx context.Context) error {
	clusterExtension := &olmv1.ClusterExtension{
		ObjectMeta: metav1.ObjectMeta{
			Name: i.ExtensionName,
		},
	}
	if err := waitForDeletion(ctx, i.config.Client, clusterExtension); err != nil {
		return fmt.Errorf("delete clusterextension %q: %w", i.ExtensionName, err)
	}
	return nil
}
