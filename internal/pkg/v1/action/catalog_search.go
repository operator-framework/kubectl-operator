package action

import (
	"context"
	"fmt"
	"time"

	catalogClient "github.com/operator-framework/kubectl-operator/internal/pkg/v1/client"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/operator-framework/kubectl-operator/pkg/action"
	olmv1 "github.com/operator-framework/operator-controller/api/v1"
	"github.com/operator-framework/operator-registry/alpha/declcfg"
)

type CatalogSearch struct {
	config            *action.Configuration
	Catalog           string
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
	var catalogList []olmv1.ClusterCatalog
	if len(i.Catalog) == 0 {
		var result olmv1.ClusterCatalogList
		listOptions := &client.ListOptions{}
		if len(i.Selector) > 0 {
			labelSelector, err := labels.Parse(i.Selector)
			if err != nil {
				return nil, fmt.Errorf("unable to parse selector %s: %v", i.Selector, err)
			}
			listOptions.LabelSelector = labelSelector
		}
		if err := i.config.Client.List(ctx, &result, listOptions); err != nil {
			return nil, err
		}
		if len(result.Items) == 0 {
			if len(i.Selector) > 0 {
				return nil, fmt.Errorf("no serving catalogs matching label selector %v found", i.Selector)
			}
			return nil, fmt.Errorf("no serving catalogs found")
		}
		for _, c := range result.Items {
			if isCatalogServing(c) {
				catalogList = append(catalogList, c)
			}
		}
	} else {
		var c olmv1.ClusterCatalog
		if err := i.config.Client.Get(ctx, types.NamespacedName{Name: i.Catalog}, &c, &client.GetOptions{}); err != nil {
			return nil, err
		}
		if isCatalogServing(c) {
			catalogList = []olmv1.ClusterCatalog{c}
		}
	}
	if len(catalogList) == 0 {
		return nil, fmt.Errorf("failed to query for catalog contents: catalog(s) unhealthy")
	}

	if len(i.Timeout) > 0 {
		if catalogListTimeout, err := time.ParseDuration(i.Timeout); err == nil {
			i.config.Config.Timeout = catalogListTimeout
		}
	}
	searchClientV1 := catalogClient.NewK8sClient(i.config.Config, i.config.Client, i.CatalogdNamespace).V1()
	catalogDeclCfg := map[string]*declcfg.DeclarativeConfig{}
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

		catalogDeclCfg[c.Name] = filterPackage(declConfigContents, i.Package)
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
