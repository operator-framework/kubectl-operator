package action

import (
	"context"
	"io"

	"github.com/operator-framework/catalogd/api/core/v1alpha1"
	"github.com/operator-framework/kubectl-operator/internal/pkg/catalogd/fetcher"
	"github.com/operator-framework/kubectl-operator/pkg/action"
	"github.com/operator-framework/operator-registry/alpha/declcfg"
)

type CatalogFetcher interface {
	FetchCatalogs(ctx context.Context, filters ...fetcher.CatalogFilterFunc) ([]v1alpha1.Catalog, error)
}

type CatalogContentStreamer interface {
	StreamCatalogContents(ctx context.Context, catalog v1alpha1.Catalog) (io.ReadCloser, error)
}

type FetcherFunc func(*action.Configuration) CatalogFetcher
type StreamerFunc func(*action.Configuration) CatalogContentStreamer

// Meta is a wrapper around the declcfg.Meta type to
// include the name of the catalog this object came from
type Meta struct {
	declcfg.Meta
	Catalog string
}
