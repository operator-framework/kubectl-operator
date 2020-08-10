package action

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-registry/pkg/image"
	"github.com/operator-framework/operator-registry/pkg/image/containerdregistry"
	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/operator-framework/kubectl-operator/internal/pkg/catalog"
)

const grpcPort = "50051"

type CatalogAdd struct {
	config *Configuration

	CatalogSourceName string
	IndexImage        string
	InjectBundles     []string
	InjectBundleMode  string
	DisplayName       string
	Publisher         string
	AddTimeout        time.Duration
	CleanupTimeout    time.Duration

	Logf            func(string, ...interface{})
	RegistryOptions []containerdregistry.RegistryOption

	registry *containerdregistry.Registry
}

func NewCatalogAdd(cfg *Configuration) *CatalogAdd {
	return &CatalogAdd{
		config: cfg,
		Logf:   func(string, ...interface{}) {},
	}
}

func (a *CatalogAdd) BindFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&a.DisplayName, "display-name", "d", "", "display name of the index")
	fs.StringVarP(&a.Publisher, "publisher", "p", "", "publisher of the index")
	fs.DurationVarP(&a.AddTimeout, "timeout", "t", time.Minute, "the amount of time to wait before cancelling the catalog addition")
	fs.DurationVar(&a.CleanupTimeout, "cleanup-timeout", time.Minute, "the amount to time to wait before cancelling cleanup")

	fs.StringArrayVarP(&a.InjectBundles, "inject-bundles", "b", nil, "inject extra bundles into the index at runtime")
	fs.StringVarP(&a.InjectBundleMode, "inject-bundle-mode", "m", "", "mode to use to inject bundles")
	_ = fs.MarkHidden("inject-bundle-mode")
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
		return nil, err
	}

	a.setDefaults(labels)

	opts := []catalog.Option{
		catalog.DisplayName(a.DisplayName),
		catalog.Publisher(a.Publisher),
	}

	if len(a.InjectBundles) == 0 {
		opts = append(opts, catalog.Image(a.IndexImage))
	}

	cs := catalog.Build(csKey, opts...)
	if err := a.createCatalogSource(ctx, cs); err != nil {
		return nil, err
	}

	var registryPod *corev1.Pod
	if len(a.InjectBundles) > 0 {
		if registryPod, err = a.createRegistryPod(ctx, cs); err != nil {
			defer a.cleanup(cs)
			return nil, err
		}

		if err := a.updateCatalogSource(ctx, cs, registryPod); err != nil {
			defer a.cleanup(cs)
			return nil, err
		}
	}

	if err := a.waitForCatalogSourceReady(ctx, cs); err != nil {
		defer a.cleanup(cs)
		return nil, err
	}

	return cs, nil
}

func (a *CatalogAdd) labelsFor(ctx context.Context, indexImage string) (map[string]string, error) {
	simpleRef := image.SimpleReference(indexImage)
	if err := a.registry.Pull(ctx, simpleRef); err != nil {
		return nil, fmt.Errorf("pull image: %v", err)
	}
	labels, err := a.registry.Labels(ctx, simpleRef)
	if err != nil {
		return nil, fmt.Errorf("get image labels: %v", err)
	}
	return labels, nil
}

func (a *CatalogAdd) setDefaults(labels map[string]string) {
	if a.DisplayName == "" {
		if v, ok := labels["operators.operatorframework.io.index.display-name"]; ok {
			a.DisplayName = v
		}
	}
	if a.Publisher == "" {
		if v, ok := labels["operators.operatorframework.io.index.publisher"]; ok {
			a.Publisher = v
		}
	}
	if a.InjectBundleMode == "" {
		if strings.HasPrefix(a.IndexImage, "quay.io/operator-framework/upstream-opm-builder") {
			a.InjectBundleMode = "semver"
		} else {
			a.InjectBundleMode = "replaces"
		}
	}
}

func (a *CatalogAdd) createCatalogSource(ctx context.Context, cs *v1alpha1.CatalogSource) error {
	if err := a.config.Client.Create(ctx, cs); err != nil {
		return fmt.Errorf("create catalogsource: %v", err)
	}
	return nil
}

func (a *CatalogAdd) createRegistryPod(ctx context.Context, cs *v1alpha1.CatalogSource) (*corev1.Pod, error) {
	command := []string{
		"/bin/sh",
		"-c",
		fmt.Sprintf(`mkdir -p /database && \
/bin/opm registry add   -d /database/index.db --mode=%s -b %s && \
/bin/opm registry serve -d /database/index.db -p %s`, a.InjectBundleMode, strings.Join(a.InjectBundles, ","), grpcPort),
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
	if err := a.config.Client.Create(ctx, pod); err != nil {
		return nil, fmt.Errorf("create registry pod: %v", err)
	}

	podKey := objectKeyForObject(pod)
	if err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		if err := a.config.Client.Get(ctx, podKey, pod); err != nil {
			return fmt.Errorf("get registry pod: %v", err)
		}
		if err := controllerutil.SetOwnerReference(cs, pod, a.config.Scheme); err != nil {
			return fmt.Errorf("set registry pod owner reference: %v", err)
		}
		if err := a.config.Client.Update(ctx, pod); err != nil {
			return fmt.Errorf("update registry pod owner reference: %v", err)
		}
		return nil
	}); err != nil {
		return nil, err
	}

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
	cs.Spec.Address = fmt.Sprintf("%s:%s", pod.Status.PodIP, grpcPort)

	injectedBundlesJSON, err := json.Marshal(a.InjectBundles)
	if err != nil {
		return fmt.Errorf("json marshal injected bundles: %v", err)
	}
	cs.ObjectMeta.Annotations = map[string]string{
		"operators.operatorframework.io/index-image":        a.IndexImage,
		"operators.operatorframework.io/inject-bundle-mode": a.InjectBundleMode,
		"operators.operatorframework.io/injected-bundles":   string(injectedBundlesJSON),
	}
	csKey := objectKeyForObject(cs)
	if err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		if err := a.config.Client.Get(ctx, csKey, cs); err != nil {
			return fmt.Errorf("get catalog source: %v", err)
		}
		if err := a.config.Client.Update(ctx, cs); err != nil {
			return fmt.Errorf("update catalog source: %v", err)
		}
		return nil
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
