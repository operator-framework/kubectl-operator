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
	config *action.Configuration

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

	CleanupTimeout time.Duration

	Logf func(string, ...interface{})

	CRDUpgradeSafetyEnforcement string
	DryRun                      string
	Output                      string
}

func NewExtensionUpdate(cfg *action.Configuration) *ExtensionUpdate {
	return &ExtensionUpdate{
		config: cfg,
		Logf:   func(string, ...interface{}) {},
	}
}

func (ou *ExtensionUpdate) Run(ctx context.Context) (*olmv1.ClusterExtension, error) {
	var ext olmv1.ClusterExtension
	var err error

	opKey := types.NamespacedName{Name: ou.ExtensionName}
	if err = ou.config.Client.Get(ctx, opKey, &ext); err != nil {
		return nil, err
	}

	if ext.Spec.Source.SourceType != olmv1.SourceTypeCatalog {
		return nil, fmt.Errorf("unrecognized source type: %q", ext.Spec.Source.SourceType)
	}

	ou.setDefaults(ext)

	if ou.Version != "" {
		if _, err = semver.ParseRange(ou.Version); err != nil {
			return nil, fmt.Errorf("failed parsing version: %w", err)
		}
	}
	if ou.Selector != "" && ou.parsedSelector == nil {
		ou.parsedSelector, err = metav1.ParseToLabelSelector(ou.Selector)
		if err != nil {
			return nil, fmt.Errorf("failed parsing selector: %w", err)
		}
	}

	constraintPolicy := olmv1.UpgradeConstraintPolicy(ou.UpgradeConstraintPolicy)
	if !ou.needsUpdate(ext, constraintPolicy) {
		return nil, ErrNoChange
	}

	ou.prepareUpdatedExtension(&ext, constraintPolicy)
	if ou.DryRun == DryRunAll {
		if err := ou.config.Client.Update(ctx, &ext, client.DryRunAll); err != nil {
			return nil, err
		}
		return &ext, nil
	}

	if err := ou.config.Client.Update(ctx, &ext); err != nil {
		return nil, err
	}

	if err := waitUntilExtensionStatusCondition(ctx, ou.config.Client, &ext, olmv1.TypeInstalled, metav1.ConditionTrue); err != nil {
		return nil, fmt.Errorf("timed out waiting for extension: %w", err)
	}

	return &ext, nil
}

func (ou *ExtensionUpdate) setDefaults(ext olmv1.ClusterExtension) {
	if !ou.IgnoreUnset {
		if ou.UpgradeConstraintPolicy == "" {
			ou.UpgradeConstraintPolicy = string(olmv1.UpgradeConstraintPolicyCatalogProvided)
		}
		if ou.CRDUpgradeSafetyEnforcement == "" {
			ou.CRDUpgradeSafetyEnforcement = string(olmv1.CRDUpgradeSafetyEnforcementStrict)
		}

		return
	}

	// IgnoreUnset is enabled
	// set all unset values to what they are on the current object
	catalogSrc := ext.Spec.Source.Catalog
	if ou.Version == "" {
		ou.Version = catalogSrc.Version
	}
	if len(ou.Channels) == 0 {
		ou.Channels = catalogSrc.Channels
	}
	if ou.UpgradeConstraintPolicy == "" {
		ou.UpgradeConstraintPolicy = string(catalogSrc.UpgradeConstraintPolicy)
	}
	if ou.CRDUpgradeSafetyEnforcement == "" {
		ou.CRDUpgradeSafetyEnforcement = string(ext.Spec.Install.Preflight.CRDUpgradeSafety.Enforcement)
	}
	if len(ou.Labels) == 0 {
		ou.Labels = ext.Labels
	}
	if ou.Selector == "" && catalogSrc.Selector != nil {
		ou.parsedSelector = catalogSrc.Selector
	}
}

func (ou *ExtensionUpdate) needsUpdate(ext olmv1.ClusterExtension, constraintPolicy olmv1.UpgradeConstraintPolicy) bool {
	catalogSrc := ext.Spec.Source.Catalog

	// object string form is used for comparison to:
	// - remove the need for potentially costly metav1.FormatLabelSelector calls
	// - avoid having to handle potential reordering of items from on cluster state
	sameSelectors := (catalogSrc.Selector == nil && ou.parsedSelector == nil) ||
		(catalogSrc.Selector != nil && ou.parsedSelector != nil &&
			catalogSrc.Selector.String() == ou.parsedSelector.String())

	if catalogSrc.Version == ou.Version &&
		slices.Equal(catalogSrc.Channels, ou.Channels) &&
		catalogSrc.UpgradeConstraintPolicy == constraintPolicy &&
		maps.Equal(ext.Labels, ou.Labels) &&
		string(ext.Spec.Install.Preflight.CRDUpgradeSafety.Enforcement) == ou.CRDUpgradeSafetyEnforcement &&
		sameSelectors {
		return false
	}

	return true
}

func (ou *ExtensionUpdate) prepareUpdatedExtension(ext *olmv1.ClusterExtension, constraintPolicy olmv1.UpgradeConstraintPolicy) {
	ext.SetLabels(ou.Labels)
	ext.Spec.Source.Catalog.Version = ou.Version
	ext.Spec.Source.Catalog.Selector = ou.parsedSelector
	ext.Spec.Source.Catalog.Channels = ou.Channels
	ext.Spec.Source.Catalog.UpgradeConstraintPolicy = constraintPolicy
	ext.Spec.Install.Preflight.CRDUpgradeSafety.Enforcement = olmv1.CRDUpgradeSafetyEnforcement(ou.CRDUpgradeSafetyEnforcement)
}
