package streamer

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/operator-framework/catalogd/api/core/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// Streamer implements the experimentalaction.CatalogContentStreamer interface
type Streamer struct {
	client corev1.CoreV1Interface
}

func New(client corev1.CoreV1Interface) *Streamer {
	return &Streamer{
		client: client,
	}
}

// StreamCatalogContents, provided a Catalog, will perform a proxy get to the catalogd HTTP server endpoint
// where the provided Catalog's content is. It will return an io.ReadCloser that can be used to read the raw
// response data from the catalogd HTTP server. It is the caller's responsibility to close it when they are
// finished reading the contents. An error is returned if any are encountered. If an error is returned the
// io.ReadCloser returned will be `nil`.
func (c *Streamer) StreamCatalogContents(ctx context.Context, catalog v1alpha1.Catalog) (io.ReadCloser, error) {
	if !meta.IsStatusConditionTrue(catalog.Status.Conditions, v1alpha1.TypeUnpacked) {
		return nil, fmt.Errorf("catalog %q is not unpacked", catalog.Name)
	}

	url, err := url.Parse(catalog.Status.ContentURL)
	if err != nil {
		return nil, fmt.Errorf("parsing catalog content url for catalog %q: %w", catalog.Name, err)
	}
	// url is expected to be in the format of
	// http://{service_name}.{namespace}.svc/{catalog_name}/all.json
	// so to get the namespace and name of the service we grab only
	// the hostname and split it on the '.' character
	ns := strings.Split(url.Hostname(), ".")[1]
	name := strings.Split(url.Hostname(), ".")[0]
	port := url.Port()
	// the ProxyGet() call below needs an explicit port value, so if
	// value from url.Port() is empty, we assume port 80.
	if port == "" {
		port = "80"
	}

	rw := c.client.Services(ns).ProxyGet(
		url.Scheme,
		name,
		port,
		url.Path,
		map[string]string{},
	)

	rc, err := rw.Stream(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting catalog contents for catalog %q: %w", catalog.Name, err)
	}
	return rc, nil
}
