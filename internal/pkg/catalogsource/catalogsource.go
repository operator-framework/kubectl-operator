package catalogsource

import (
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type Option func(*v1alpha1.CatalogSource)

func DisplayName(v string) Option {
	return func(cs *v1alpha1.CatalogSource) {
		cs.Spec.DisplayName = v
	}
}
func Image(v string) Option {
	return func(cs *v1alpha1.CatalogSource) {
		cs.Spec.Image = v
	}
}

func Publisher(v string) Option {
	return func(cs *v1alpha1.CatalogSource) {
		cs.Spec.Publisher = v
	}
}

func GRPCPodConfig(grpcPodConfig *v1alpha1.GrpcPodConfig) Option {
	return func(cs *v1alpha1.CatalogSource) {
		cs.Spec.GrpcPodConfig = grpcPodConfig
	}
}

func Build(key types.NamespacedName, opts ...Option) *v1alpha1.CatalogSource {
	cs := &v1alpha1.CatalogSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
		},
		Spec: v1alpha1.CatalogSourceSpec{
			SourceType: v1alpha1.SourceTypeGrpc,
		},
	}
	for _, o := range opts {
		o(cs)
	}
	return cs
}
