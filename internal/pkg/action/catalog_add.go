package action

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/containerd/containerd/archive/compression"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/platforms"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-registry/pkg/image"
	"github.com/operator-framework/operator-registry/pkg/image/containerdregistry"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/operator-framework/kubectl-operator/internal/pkg/catalogsource"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

const (
	grpcPort              = "50051"
	dbPathLabel           = "operators.operatorframework.io.index.database.v1"
	alphaDisplayNameLabel = "alpha.operators.operatorframework.io.index.display-name.v1"
	alphaPublisherLabel   = "alpha.operators.operatorframework.io.index.publisher.v1"
	defaultDatabasePath   = "/database/index.db"
)

type CatalogAdd struct {
	config *action.Configuration

	CatalogSourceName string
	IndexImage        string
	InjectBundles     []string
	InjectBundleMode  string
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
	}

	if len(a.InjectBundles) == 0 {
		opts = append(opts, catalogsource.Image(a.IndexImage))
	}

	cs := catalogsource.Build(csKey, opts...)
	if err := a.config.Client.Create(ctx, cs); err != nil {
		return nil, fmt.Errorf("create catalogsource: %v", err)
	}

	var registryPod *corev1.Pod
	if len(a.InjectBundles) > 0 {
		dbPath, ok := labels[dbPathLabel]
		if !ok {
			// No database path label, so assume this is an index base image.
			// Choose "semver" bundle add mode (if not explicitly set) and
			// use the default database path.
			if a.InjectBundleMode == "" {
				a.InjectBundleMode = "semver"
			}
			dbPath = defaultDatabasePath
		}
		if a.InjectBundleMode == "" {
			a.InjectBundleMode = "replaces"
		}
		if registryPod, err = a.createRegistryPod(ctx, cs, dbPath); err != nil {
			defer a.cleanup(cs)
			return nil, err
		}

		if err := a.updateCatalogSource(ctx, cs, registryPod); err != nil {
			defer a.cleanup(cs)
			return nil, fmt.Errorf("update catalog source: %v", err)
		}
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

func (a *CatalogAdd) createRegistryPod(ctx context.Context, cs *v1alpha1.CatalogSource, dbPath string) (*corev1.Pod, error) {
	dbDir := path.Dir(dbPath)
	command := []string{
		"/bin/sh",
		"-c",
		fmt.Sprintf(`mkdir -p %s && \
/bin/opm registry add   -d %s --mode=%s -b %s && \
/bin/opm registry serve -d %s -p %s`, dbDir, dbPath, a.InjectBundleMode, strings.Join(a.InjectBundles, ","), dbPath, grpcPort),
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", a.CatalogSourceName, rand.String(4)),
			Namespace: a.config.Namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "registry",
					Image:   a.IndexImage,
					Command: command,
				},
			},
		},
	}

	if err := controllerutil.SetOwnerReference(cs, pod, a.config.Scheme); err != nil {
		return nil, fmt.Errorf("set registry pod owner reference: %v", err)
	}
	if err := a.config.Client.Create(ctx, pod); err != nil {
		return nil, fmt.Errorf("create registry pod: %v", err)
	}

	podKey := objectKeyForObject(pod)
	if err := wait.PollImmediateUntil(time.Millisecond*250, func() (bool, error) {
		if err := a.config.Client.Get(ctx, podKey, pod); err != nil {
			return false, err
		}
		if pod.Status.Phase == corev1.PodRunning && pod.Status.PodIP != "" {
			return true, nil
		}
		return false, nil
	}, ctx.Done()); err != nil {
		return nil, fmt.Errorf("registry pod not ready: %v", err)
	}
	return pod, nil
}

func (a *CatalogAdd) updateCatalogSource(ctx context.Context, cs *v1alpha1.CatalogSource, pod *corev1.Pod) error {
	injectedBundlesJSON, err := json.Marshal(a.InjectBundles)
	if err != nil {
		return fmt.Errorf("json marshal injected bundles: %v", err)
	}

	csKey := objectKeyForObject(cs)
	if err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		if err := a.config.Client.Get(ctx, csKey, cs); err != nil {
			return fmt.Errorf("get catalog source: %v", err)
		}

		cs.Spec.Address = fmt.Sprintf("%s:%s", pod.Status.PodIP, grpcPort)
		cs.ObjectMeta.Annotations = map[string]string{
			"operators.operatorframework.io/index-image":        a.IndexImage,
			"operators.operatorframework.io/inject-bundle-mode": a.InjectBundleMode,
			"operators.operatorframework.io/injected-bundles":   string(injectedBundlesJSON),
		}

		return a.config.Client.Update(ctx, cs)
	}); err != nil {
		return err
	}
	return nil
}

func (a *CatalogAdd) waitForCatalogSourceReady(ctx context.Context, cs *v1alpha1.CatalogSource) error {
	csKey := objectKeyForObject(cs)
	if err := wait.PollImmediateUntil(time.Millisecond*250, func() (bool, error) {
		if err := a.config.Client.Get(ctx, csKey, cs); err != nil {
			return false, err
		}
		if cs.Status.GRPCConnectionState != nil {
			if cs.Status.GRPCConnectionState.LastObservedState == "READY" {
				return true, nil
			}
		}
		return false, nil
	}, ctx.Done()); err != nil {
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
