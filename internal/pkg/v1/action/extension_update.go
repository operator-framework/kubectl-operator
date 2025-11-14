package action

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/blang/semver/v4"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	"github.com/operator-framework/kubectl-operator/pkg/action"
)

type ExtensionUpdate struct {
	config        *action.Configuration
	ExtensionName string

	Version  string
	Channels []string
	Selector string
	// parsedSelector is used internally to avoid potentially costly transformations
	// between string and metav1.LabelSelector formats
	parsedSelector          *metav1.LabelSelector
	UpgradeConstraintPolicy string
	Labels                  map[string]string
	IgnoreUnset             bool

	CleanupTimeout              time.Duration
	CRDUpgradeSafetyEnforcement string

	DryRun string
	Output string
	Logf   func(string, ...interface{})
}

func NewExtensionUpdate(cfg *action.Configuration) *ExtensionUpdate {
	return &ExtensionUpdate{
		config: cfg,
		Logf:   func(string, ...interface{}) {},
	}
}

func (i *ExtensionUpdate) Run(ctx context.Context) (*olmv1.ClusterExtension, error) {
	var ext olmv1.ClusterExtension
	var err error

	opKey := types.NamespacedName{Name: i.ExtensionName}
	if err = i.config.Client.Get(ctx, opKey, &ext); err != nil {
		return nil, err
	}

	if ext.Spec.Source.SourceType != olmv1.SourceTypeCatalog {
		return nil, fmt.Errorf("unrecognized source type: %q", ext.Spec.Source.SourceType)
	}

	i.setDefaults(ext)

	if i.Version != "" {
		if _, err = semver.ParseRange(i.Version); err != nil {
			return nil, fmt.Errorf("failed parsing version: %w", err)
		}
	}
	if i.Selector != "" && i.parsedSelector == nil {
		i.parsedSelector, err = metav1.ParseToLabelSelector(i.Selector)
		if err != nil {
			return nil, fmt.Errorf("failed parsing selector: %w", err)
		}
	}

	constraintPolicy := olmv1.UpgradeConstraintPolicy(i.UpgradeConstraintPolicy)
	if !i.needsUpdate(ext, constraintPolicy) {
		return nil, ErrNoChange
	}

	i.prepareUpdatedExtension(&ext, constraintPolicy)
	if i.DryRun == DryRunAll {
		if err := i.config.Client.Update(ctx, &ext, client.DryRunAll); err != nil {
			return nil, err
		}
		return &ext, nil
	}

	if err := i.config.Client.Update(ctx, &ext); err != nil {
		return nil, err
	}

	if err := waitUntilExtensionStatusCondition(ctx, i.config.Client, &ext, olmv1.TypeInstalled, metav1.ConditionTrue); err != nil {
		return nil, fmt.Errorf("timed out waiting for extension: %w", err)
	}

	return &ext, nil
}

func (i *ExtensionUpdate) setDefaults(ext olmv1.ClusterExtension) {
	if !i.IgnoreUnset {
		if i.UpgradeConstraintPolicy == "" {
			i.UpgradeConstraintPolicy = string(olmv1.UpgradeConstraintPolicyCatalogProvided)
		}
		if i.CRDUpgradeSafetyEnforcement == "" {
			i.CRDUpgradeSafetyEnforcement = string(olmv1.CRDUpgradeSafetyEnforcementStrict)
		}

		return
	}

	// IgnoreUnset is enabled
	// set all unset values to what they are on the current object
	catalogSrc := ext.Spec.Source.Catalog
	if i.Version == "" {
		i.Version = catalogSrc.Version
	}
	if len(i.Channels) == 0 {
		i.Channels = catalogSrc.Channels
	}
	if i.UpgradeConstraintPolicy == "" {
		i.UpgradeConstraintPolicy = string(catalogSrc.UpgradeConstraintPolicy)
	}
	if i.CRDUpgradeSafetyEnforcement == "" {
		i.CRDUpgradeSafetyEnforcement = string(ext.Spec.Install.Preflight.CRDUpgradeSafety.Enforcement)
	}
	if len(i.Labels) == 0 {
		i.Labels = ext.Labels
	}
	if i.Selector == "" && catalogSrc.Selector != nil {
		i.parsedSelector = catalogSrc.Selector
	}
}

func (i *ExtensionUpdate) needsUpdate(ext olmv1.ClusterExtension, constraintPolicy olmv1.UpgradeConstraintPolicy) bool {
	catalogSrc := ext.Spec.Source.Catalog

	// object string form is used for comparison to:
	// - remove the need for potentially costly metav1.FormatLabelSelector calls
	// - avoid having to handle potential reordering of items from on cluster state
	sameSelectors := (catalogSrc.Selector == nil && i.parsedSelector == nil) ||
		(catalogSrc.Selector != nil && i.parsedSelector != nil &&
			catalogSrc.Selector.String() == i.parsedSelector.String())

	if catalogSrc.Version == i.Version &&
		slices.Equal(catalogSrc.Channels, i.Channels) &&
		catalogSrc.UpgradeConstraintPolicy == constraintPolicy &&
		maps.Equal(ext.Labels, i.Labels) &&
		string(ext.Spec.Install.Preflight.CRDUpgradeSafety.Enforcement) == i.CRDUpgradeSafetyEnforcement &&
		sameSelectors {
		return false
	}

	return true
}

func (i *ExtensionUpdate) prepareUpdatedExtension(ext *olmv1.ClusterExtension, constraintPolicy olmv1.UpgradeConstraintPolicy) {
	ext.SetLabels(i.Labels)
	ext.Spec.Source.Catalog.Version = i.Version
	ext.Spec.Source.Catalog.Selector = i.parsedSelector
	ext.Spec.Source.Catalog.Channels = i.Channels
	ext.Spec.Source.Catalog.UpgradeConstraintPolicy = constraintPolicy
	ext.Spec.Install.Preflight.CRDUpgradeSafety.Enforcement = olmv1.CRDUpgradeSafetyEnforcement(i.CRDUpgradeSafetyEnforcement)
}
