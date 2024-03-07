package fetcher

import (
	"context"
	"errors"

	"github.com/operator-framework/catalogd/api/core/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CatalogFilterFunc is a function used to filter catalogs
// returned by Fetcher.FetchCatalogs. A return value of `true`
// signals that a catalog meets the filter criteria and should
// be included in the returned set of catalogs
type CatalogFilterFunc func(catalog *v1alpha1.Catalog) bool

func New(client client.Client) *Fetcher {
	return &Fetcher{
		client: client,
	}
}

// Fetcher is an implementation of the experimentalaction.CatalogFetcher interface
type Fetcher struct {
	client client.Client
}

// FetchCatalogs will retrieve a list of catalogs on the cluster and evaluate each one against the
// set of catalog filter functions. The returned set of catalogs will be all the catalogs that
// meet the filtering criteria.
func (c *Fetcher) FetchCatalogs(ctx context.Context, filters ...CatalogFilterFunc) ([]v1alpha1.Catalog, error) {
	if c.client == nil {
		return nil, errors.New("nil client provided - failing early")
	}
	catalogList := &v1alpha1.CatalogList{}
	err := c.client.List(ctx, catalogList)
	if err != nil {
		return nil, err
	}

	catalogs := []v1alpha1.Catalog{}
	for _, catalog := range catalogList.Items {
		filteredOut := false
		for _, filter := range filters {
			if !filter(catalog.DeepCopy()) {
				filteredOut = true
			}
		}

		if filteredOut {
			continue
		}

		catalogs = append(catalogs, catalog)
	}

	return catalogs, nil
}

// WithNameFilter is a helper for filtering catalogs
// that match the given name
func WithNameFilter(name string) CatalogFilterFunc {
	return func(catalog *v1alpha1.Catalog) bool {
		if name == "" {
			return true
		}
		return catalog.Name == name
	}
}

// WithUnpackedFilter is a helper for filtering
// catalogs that have the status condition "Unpacked"
// set to "True"
func WithUnpackedFilter() CatalogFilterFunc {
	return func(catalog *v1alpha1.Catalog) bool {
		return meta.IsStatusConditionTrue(catalog.Status.Conditions, v1alpha1.TypeUnpacked)
	}
}
