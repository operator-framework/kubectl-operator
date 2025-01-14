package v1

import (
	"context"
	"fmt"
	"github.com/operator-framework/kubectl-operator/pkg/action"
	ocv1 "github.com/operator-framework/operator-controller/api/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type OperatorUninstall struct {
	config *action.Configuration

	Package string

	Logf func(string, ...interface{})
}

func NewOperatorUninstall(cfg *action.Configuration) *OperatorUninstall {
	return &OperatorUninstall{
		config: cfg,
		Logf:   func(string, ...interface{}) {},
	}
}

type ErrPackageNotFound struct {
	PackageName string
}

func (e ErrPackageNotFound) Error() string {
	return fmt.Sprintf("package %q not found", e.PackageName)
}

func (u *OperatorUninstall) Run(ctx context.Context) error {
	clusterExtension := &ocv1.ClusterExtension{}
	if err := u.config.Client.Get(ctx, types.NamespacedName{Name: u.Package}, clusterExtension); err != nil {
		if apierrors.IsNotFound(err) {
			return &ErrPackageNotFound{u.Package}
		}
		return fmt.Errorf("get clusterextension %q: %v", u.Package, err)
	}

	namespaceName := clusterExtension.Spec.Namespace
	saName := clusterExtension.Spec.ServiceAccount.Name

	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("kubectl-operator-%s-cluster-admin", saName),
		},
	}
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespaceName,
			Name:      saName,
		},
	}
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName,
		},
	}
	if err := deleteAndWait(ctx, u.config.Client, clusterExtension); err != nil {
		return fmt.Errorf("delete clusterextension %q: %v", u.Package, err)
	}
	return deleteAndWait(ctx, u.config.Client, clusterRoleBinding, serviceAccount, namespace)
}
