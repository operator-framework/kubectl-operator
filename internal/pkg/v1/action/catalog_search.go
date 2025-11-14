package action

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"
	"github.com/operator-framework/operator-registry/alpha/declcfg"

	catalogClient "github.com/operator-framework/kubectl-operator/internal/pkg/v1/client"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

type CatalogSearch struct {
	config      *action.Configuration
	CatalogName string

	OutputFormat      string
	Selector          string
	ListVersions      bool
	Package           string
	CatalogdNamespace string
	Timeout           string

	Logf func(string, ...interface{})
}

func NewCatalogSearch(cfg *action.Configuration) *CatalogSearch {
	return &CatalogSearch{
		config: cfg,
		Logf:   func(string, ...interface{}) {},
	}
}

func (i *CatalogSearch) Run(ctx context.Context) (map[string]*declcfg.DeclarativeConfig, error) {
	if len(i.Timeout) > 0 {
		catalogListTimeout, err := time.ParseDuration(i.Timeout)
		if err != nil {
			return nil, fmt.Errorf("failed to parse timeout %s: %w", i.Timeout, err)
		}
		i.config.Config.Timeout = catalogListTimeout
	}
	var catalogList []olmv1.ClusterCatalog
	listCmd := NewCatalogInstalledGet(i.config)
	listCmd.Selector = i.Selector
	listCmd.CatalogName = i.CatalogName
	result, err := listCmd.Run(ctx)
	if err != nil {
		return nil, err
	}
	for _, c := range result {
		if isCatalogServing(c) {
			catalogList = append(catalogList, c)
		}
	}
	if len(catalogList) == 0 {
		if len(i.CatalogName) != 0 {
			return nil, fmt.Errorf("failed to query for catalog contents: catalog(s) unhealthy")
		}
		if len(i.Selector) > 0 {
			return nil, fmt.Errorf("no serving catalogs matching label selector %v found", i.Selector)
		}
		return nil, fmt.Errorf("no serving catalogs found")
	}
	searchClientV1 := catalogClient.NewK8sClient(i.config.Config, i.config.Client, i.CatalogdNamespace).V1()
	catalogDeclCfg := map[string]*declcfg.DeclarativeConfig{}
	foundPackage := len(i.Package) == 0 // whether to check for empty package query
	for _, c := range catalogList {
		catalogContent, err := searchClientV1.All(ctx, &c)
		if err != nil {
			return nil, err
		}
		defer catalogContent.Close()
		declConfigContents, err := declcfg.LoadReader(catalogContent)
		if err != nil {
			return nil, err
		}
		if len(i.Package) == 0 {
			catalogDeclCfg[c.Name] = declConfigContents
			continue
		}

		filteredContents := filterPackage(declConfigContents, i.Package)

		if len(filteredContents.Packages) > 0 {
			catalogDeclCfg[c.Name] = filteredContents
			foundPackage = true
		}
	}
	if !foundPackage {
		// package name was specified and query was empty across all available catalogs.
		if len(i.CatalogName) != 0 {
			return nil, fmt.Errorf("package %s was not found in ClusterCatalog %s", i.Package, i.CatalogName)
		}
		if len(i.Selector) > 0 {
			return nil, fmt.Errorf("package %s was not found in ClusterCatalogs matching label %s", i.Package, i.Selector)
		}
		return nil, fmt.Errorf("package %s was not found in any serving ClusterCatalog", i.Package)
	}
	return catalogDeclCfg, nil
}

func isCatalogServing(c olmv1.ClusterCatalog) bool {
	if c.Spec.AvailabilityMode != olmv1.AvailabilityModeAvailable {
		return false
	}
	if !meta.IsStatusConditionPresentAndEqual(c.Status.Conditions, olmv1.TypeServing, metav1.ConditionTrue) {
		return false
	}
	if c.Status.ResolvedSource == nil || c.Status.ResolvedSource.Image == nil {
		return false
	}
	return true
}

func filterPackage(dcfg *declcfg.DeclarativeConfig, packageName string) *declcfg.DeclarativeConfig {
	filteredDeclCfg := &declcfg.DeclarativeConfig{
		Channels:     []declcfg.Channel{},
		Bundles:      []declcfg.Bundle{},
		Deprecations: []declcfg.Deprecation{},
		Others:       []declcfg.Meta{},
	}
	for _, p := range dcfg.Packages {
		if p.Name == packageName {
			filteredDeclCfg.Packages = []declcfg.Package{p}
			break
		}
	}
	for _, e := range dcfg.Channels {
		if e.Package == packageName {
			filteredDeclCfg.Channels = append(filteredDeclCfg.Channels, e)
		}
	}

	for _, e := range dcfg.Bundles {
		if e.Package == packageName {
			filteredDeclCfg.Bundles = append(filteredDeclCfg.Bundles, e)
		}
	}

	for _, e := range dcfg.Deprecations {
		if e.Package == packageName {
			filteredDeclCfg.Deprecations = append(filteredDeclCfg.Deprecations, e)
		}
	}

	for _, e := range dcfg.Others {
		if e.Package == packageName {
			filteredDeclCfg.Others = append(filteredDeclCfg.Others, e)
		}
	}
	return filteredDeclCfg
}
