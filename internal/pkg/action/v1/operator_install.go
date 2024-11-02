package v1

import (
	"context"
	"fmt"
	"time"

	"errors"
	"github.com/operator-framework/kubectl-operator/pkg/action"
	ocv1 "github.com/operator-framework/operator-controller/api/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	applyconfigurationscorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	applyconfigurationsrbacv1 "k8s.io/client-go/applyconfigurations/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type OperatorInstall struct {
	config *action.Configuration

	UnsafeCreateClusterRoleBinding bool

	Namespace      OperatorInstallNamespaceConfig
	ServiceAccount string

	Package         string
	Channels        []string
	Version         string
	CatalogSelector metav1.LabelSelector

	CleanupTimeout time.Duration

	Logf func(string, ...interface{})
}

type OperatorInstallNamespaceConfig struct {
	Name        string
	Labels      map[string]string
	Annotations map[string]string
}

func NewOperatorInstall(cfg *action.Configuration) *OperatorInstall {
	return &OperatorInstall{
		config: cfg,
		Logf:   func(string, ...interface{}) {},
	}
}

func (i *OperatorInstall) applyNamespace(ctx context.Context) error {
	ac := applyconfigurationscorev1.Namespace(i.Namespace.Name)
	if i.Namespace.Labels != nil {
		ac = ac.WithLabels(i.Namespace.Labels)
	}
	if i.Namespace.Annotations != nil {
		ac = ac.WithAnnotations(i.Namespace.Annotations)
	}
	return patchObject(ctx, i.config.Client, ac)
}

func (i *OperatorInstall) applyServiceAccount(ctx context.Context) error {
	ac := applyconfigurationscorev1.ServiceAccount(i.ServiceAccount, i.Namespace.Name)
	return patchObject(ctx, i.config.Client, ac)
}

func (i *OperatorInstall) applyClusterRoleBinding(ctx context.Context, clusterRoleName string) error {
	name := fmt.Sprintf("kubectl-operator-%s-cluster-admin", i.ServiceAccount)
	ac := applyconfigurationsrbacv1.ClusterRoleBinding(name).
		WithSubjects(applyconfigurationsrbacv1.Subject().WithNamespace(i.Namespace.Name).WithKind("ServiceAccount").WithName(i.ServiceAccount)).
		WithRoleRef(applyconfigurationsrbacv1.RoleRef().WithKind("ClusterRole").WithName(clusterRoleName))
	return patchObject(ctx, i.config.Client, ac)
}

func (i *OperatorInstall) applyClusterExtension(ctx context.Context) error {
	catalogSource := map[string]interface{}{
		"packageName": i.Package,
	}
	if i.Version != "" {
		catalogSource["version"] = i.Version
	}
	if i.Channels != nil {
		catalogSource["channels"] = i.Channels
	}
	if i.CatalogSelector.MatchLabels != nil || i.CatalogSelector.MatchExpressions != nil {
		catalogSource["selector"] = i.CatalogSelector
	}
	u := unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": ocv1.GroupVersion.String(),
		"kind":       "ClusterExtension",
		"metadata": map[string]interface{}{
			"name": i.Package,
		},
		"spec": map[string]interface{}{
			"namespace": i.Namespace.Name,
			"serviceAccount": map[string]interface{}{
				"name": i.ServiceAccount,
			},
			"source": map[string]interface{}{
				"sourceType": "Catalog",
				"catalog":    catalogSource,
			},
		},
	}}
	return patchObject(ctx, i.config.Client, &u)
}

func (i *OperatorInstall) Run(ctx context.Context) (*ocv1.ClusterExtension, error) {
	if err := i.applyNamespace(ctx); err != nil {
		return nil, fmt.Errorf("apply namespace %q: %v", i.Namespace.Name, err)
	}
	if err := i.applyServiceAccount(ctx); err != nil {
		return nil, fmt.Errorf("apply service account %q: %v", i.ServiceAccount, err)
	}

	if i.UnsafeCreateClusterRoleBinding {
		if err := i.applyClusterRoleBinding(ctx, "cluster-admin"); err != nil {
			return nil, fmt.Errorf("apply cluster role binding: %v", err)
		}
	}

	if err := i.applyClusterExtension(ctx); err != nil {
		return nil, fmt.Errorf("apply cluster extension: %v", err)
	}

	clusterExtension, err := i.waitForClusterExtensionInstalled(ctx)
	if err != nil {
		cleanupCtx, cancelCleanup := context.WithTimeout(context.Background(), i.CleanupTimeout)
		defer cancelCleanup()
		cleanupErr := i.cleanup(cleanupCtx)
		return nil, errors.Join(err, cleanupErr)
	}
	return clusterExtension, nil
}

func (i *OperatorInstall) waitForClusterExtensionInstalled(ctx context.Context) (*ocv1.ClusterExtension, error) {
	clusterExtension := &ocv1.ClusterExtension{
		ObjectMeta: metav1.ObjectMeta{
			Name: i.Package,
		},
	}
	errMsg := ""
	key := client.ObjectKeyFromObject(clusterExtension)
	if err := wait.PollUntilContextCancel(ctx, time.Millisecond*250, true, func(conditionCtx context.Context) (bool, error) {
		if err := i.config.Client.Get(conditionCtx, key, clusterExtension); err != nil {
			return false, err
		}
		progressingCondition := meta.FindStatusCondition(clusterExtension.Status.Conditions, ocv1.TypeProgressing)
		if progressingCondition != nil && progressingCondition.Reason != ocv1.ReasonSucceeded {
			errMsg = progressingCondition.Message
			return false, nil
		}
		if !meta.IsStatusConditionPresentAndEqual(clusterExtension.Status.Conditions, ocv1.TypeInstalled, metav1.ConditionTrue) {
			return false, nil
		}
		return true, nil
	}); err != nil {
		if errMsg == "" {
			errMsg = err.Error()
		}
		return nil, fmt.Errorf("cluster extension %q did not finish installing: %s", clusterExtension.Name, errMsg)
	}
	return clusterExtension, nil
}

func (i *OperatorInstall) cleanup(ctx context.Context) error {
	clusterExtension := &ocv1.ClusterExtension{
		ObjectMeta: metav1.ObjectMeta{
			Name: i.Package,
		},
	}
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("kubectl-operator-%s-cluster-admin", i.ServiceAccount),
		},
	}
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: i.Namespace.Name,
			Name:      i.ServiceAccount,
		},
	}
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: i.Namespace.Name,
		},
	}
	if err := deleteAndWait(ctx, i.config.Client, clusterExtension); err != nil {
		return fmt.Errorf("delete clusterextension %q: %v", i.Package, err)
	}
	return deleteAndWait(ctx, i.config.Client, clusterRoleBinding, serviceAccount, namespace)
}
