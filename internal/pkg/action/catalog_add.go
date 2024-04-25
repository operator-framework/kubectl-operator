package action

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/containerd/containerd/archive/compression"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/platforms"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-registry/pkg/image"
	"github.com/operator-framework/operator-registry/pkg/image/containerdregistry"

	"github.com/operator-framework/kubectl-operator/internal/pkg/catalogsource"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

const (
	alphaDisplayNameLabel = "alpha.operators.operatorframework.io.index.display-name.v1"
	alphaPublisherLabel   = "alpha.operators.operatorframework.io.index.publisher.v1"
)

type CatalogAdd struct {
	config *action.Configuration

	CatalogSourceName string
	IndexImage        string
	DisplayName       string
	Publisher         string
	CleanupTimeout    time.Duration

	Logf            func(string, ...interface{})
	RegistryOptions []containerdregistry.RegistryOption

	registry *containerdregistry.Registry
}

func NewCatalogAdd(cfg *action.Configuration) *CatalogAdd {
	return &CatalogAdd{
		config: cfg,
		Logf:   func(string, ...interface{}) {},
	}
}

func (a *CatalogAdd) Run(ctx context.Context) (*v1alpha1.CatalogSource, error) {
	var err error
	a.registry, err = containerdregistry.NewRegistry(a.RegistryOptions...)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := a.registry.Destroy(); err != nil {
			a.Logf("registry cleanup: %v", err)
		}
	}()

	csKey := types.NamespacedName{
		Namespace: a.config.Namespace,
		Name:      a.CatalogSourceName,
	}

	labels, err := a.labelsFor(ctx, a.IndexImage)
	if err != nil {
		return nil, fmt.Errorf("get image labels: %v", err)
	}

	a.setDefaults(labels)

	opts := []catalogsource.Option{
		catalogsource.DisplayName(a.DisplayName),
		catalogsource.Publisher(a.Publisher),
		catalogsource.Image(a.IndexImage),
	}

	cs := catalogsource.Build(csKey, opts...)
	if err := a.config.Client.Create(ctx, cs); err != nil {
		return nil, fmt.Errorf("create catalogsource: %v", err)
	}

	if err := a.waitForCatalogSourceReady(ctx, cs); err != nil {
		defer a.cleanup(cs)
		return nil, err
	}

	return cs, nil
}

func (a *CatalogAdd) labelsFor(ctx context.Context, indexImage string) (map[string]string, error) {
	ref := image.SimpleReference(indexImage)
	if err := a.registry.Pull(ctx, ref); err != nil {
		return nil, fmt.Errorf("pull image: %v", err)
	}

	ctx = namespaces.WithNamespace(ctx, namespaces.Default)
	img, err := a.registry.Images().Get(ctx, ref.String())
	if err != nil {
		return nil, fmt.Errorf("get image from local registry: %v", err)
	}

	manifest, err := images.Manifest(ctx, a.registry.Content(), img.Target, platforms.All)
	if err != nil {
		return nil, fmt.Errorf("resolve image manifest: %v", err)
	}

	ra, err := a.registry.Content().ReaderAt(ctx, manifest.Config)
	if err != nil {
		return nil, fmt.Errorf("get image reader: %v", err)
	}
	defer ra.Close()

	decompressed, err := compression.DecompressStream(io.NewSectionReader(ra, 0, ra.Size()))
	if err != nil {
		return nil, fmt.Errorf("decompress image data: %v", err)
	}
	var imageMeta ocispec.Image
	dec := json.NewDecoder(decompressed)
	if err := dec.Decode(&imageMeta); err != nil {
		return nil, fmt.Errorf("decode image metadata: %v", err)
	}
	return imageMeta.Config.Labels, nil
}

func (a *CatalogAdd) setDefaults(labels map[string]string) {
	if a.DisplayName == "" {
		if v, ok := labels[alphaDisplayNameLabel]; ok {
			a.DisplayName = v
		}
	}
	if a.Publisher == "" {
		if v, ok := labels[alphaPublisherLabel]; ok {
			a.Publisher = v
		}
	}
}

func (a *CatalogAdd) waitForCatalogSourceReady(ctx context.Context, cs *v1alpha1.CatalogSource) error {
	csKey := objectKeyForObject(cs)
	if err := wait.PollUntilContextCancel(ctx, time.Millisecond*250, true, func(conditionCtx context.Context) (bool, error) {
		if err := a.config.Client.Get(conditionCtx, csKey, cs); err != nil {
			return false, err
		}
		if cs.Status.GRPCConnectionState != nil {
			if cs.Status.GRPCConnectionState.LastObservedState == "READY" {
				return true, nil
			}
		}
		return false, nil
	}); err != nil {
		return fmt.Errorf("catalogsource connection not ready: %v", err)
	}
	return nil
}

func (a *CatalogAdd) cleanup(cs *v1alpha1.CatalogSource) {
	ctx, cancel := context.WithTimeout(context.Background(), a.CleanupTimeout)
	defer cancel()
	if err := a.config.Client.Delete(ctx, cs); err != nil && !apierrors.IsNotFound(err) {
		a.Logf("delete catalogsource %q: %v", cs.Name, err)
	}
}
