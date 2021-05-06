package action_test

import (
	"context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "github.com/operator-framework/api/pkg/operators/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/operator-framework/kubectl-operator/pkg/action"
)

var _ = Describe("OperatorListOperands", func() {
	var (
		cfg          action.Configuration
		operator     *v1.Operator
		csv          *v1alpha1.ClusterServiceVersion
		crd          *apiextv1.CustomResourceDefinition
		og           *v1.OperatorGroup
		etcdcluster1 *unstructured.Unstructured
		etcdcluster2 *unstructured.Unstructured
		etcdcluster3 *unstructured.Unstructured
	)

	BeforeEach(func() {
		sch, err := action.NewScheme()
		Expect(err).To(BeNil())

		etcdclusterGVK := schema.GroupVersionKind{
			Group:   "etcd.database.coreos.com",
			Version: "v1beta2",
			Kind:    "EtcdCluster",
		}

		sch.AddKnownTypeWithName(etcdclusterGVK, &unstructured.Unstructured{})
		sch.AddKnownTypeWithName(schema.GroupVersionKind{
			Group:   "etcd.database.coreos.com",
			Version: "v1beta2",
			Kind:    "EtcdClusterList",
		}, &unstructured.UnstructuredList{})

		operator = &v1.Operator{
			ObjectMeta: metav1.ObjectMeta{Name: "etcd.etcd-namespace"},
			Status: v1.OperatorStatus{
				Components: &v1.Components{
					Refs: []v1.RichReference{
						{
							ObjectReference: &corev1.ObjectReference{
								APIVersion: "operators.coreos.com/v1alpha1",
								Kind:       "ClusterServiceVersion",
								Name:       "etcdoperator.v0.9.4-clusterwide",
								Namespace:  "etcd-namespace",
							},
						},
					},
				},
			},
		}

		csv = &v1alpha1.ClusterServiceVersion{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "etcdoperator.v0.9.4-clusterwide",
				Namespace: "etcd-namespace",
			},
			Spec: v1alpha1.ClusterServiceVersionSpec{
				CustomResourceDefinitions: v1alpha1.CustomResourceDefinitions{
					Owned: []v1alpha1.CRDDescription{
						{
							Name:    "etcdclusters.etcd.database.coreos.com",
							Version: "v1beta2",
							Kind:    "EtcdCluster",
						},
					},
				},
			},
			Status: v1alpha1.ClusterServiceVersionStatus{Phase: v1alpha1.CSVPhaseSucceeded},
		}

		og = &v1.OperatorGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "etcd",
				Namespace: "etcd-namespace",
			},
			Status: v1.OperatorGroupStatus{Namespaces: []string{""}},
		}

		crd = &apiextv1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: "etcdclusters.etcd.database.coreos.com",
			},
			Spec: apiextv1.CustomResourceDefinitionSpec{
				Group: "etcd.database.coreos.com",
				Names: apiextv1.CustomResourceDefinitionNames{
					ListKind: "EtcdClusterList",
				},
			},
		}
		etcdcluster1 = &unstructured.Unstructured{}
		etcdcluster1.SetGroupVersionKind(etcdclusterGVK)
		etcdcluster1.SetNamespace("ns1")
		etcdcluster1.SetName("cluster1")

		etcdcluster2 = &unstructured.Unstructured{}
		etcdcluster2.SetGroupVersionKind(etcdclusterGVK)
		etcdcluster2.SetNamespace("ns2")
		etcdcluster2.SetName("cluster2")

		etcdcluster3 = &unstructured.Unstructured{}
		etcdcluster3.SetGroupVersionKind(etcdclusterGVK)
		// Empty namespace to simulate cluster-scoped object.
		etcdcluster3.SetNamespace("")
		etcdcluster3.SetName("cluster3")

		cl := fake.NewClientBuilder().
			WithObjects(operator, csv, og, crd, etcdcluster1, etcdcluster2, etcdcluster3).
			WithScheme(sch).
			Build()
		cfg.Scheme = sch
		cfg.Client = cl
		cfg.Namespace = "etcd-namespace"
	})

	It("should fail due to missing operator", func() {
		lister := action.NewOperatorListOperands(&cfg)
		_, err := lister.Run(context.TODO(), "missing")
		Expect(err.Error()).To(ContainSubstring("package missing.etcd-namespace not found"))
	})

	It("should fail due to missing operator components", func() {
		operator.Status.Components = nil
		Expect(cfg.Client.Update(context.TODO(), operator)).To(Succeed())

		lister := action.NewOperatorListOperands(&cfg)
		_, err := lister.Run(context.TODO(), "etcd")
		Expect(err.Error()).To(ContainSubstring("could not find underlying components for operator"))
	})

	It("should fail due to missing CSV in operator components", func() {
		operator.Status.Components = &v1.Components{}
		Expect(cfg.Client.Update(context.TODO(), operator)).To(Succeed())

		lister := action.NewOperatorListOperands(&cfg)
		_, err := lister.Run(context.TODO(), "etcd")
		Expect(err.Error()).To(ContainSubstring("could not find underlying CSV for operator"))
	})

	It("should fail due to missing CSV in cluster", func() {
		Expect(cfg.Client.Delete(context.TODO(), csv)).To(Succeed())

		lister := action.NewOperatorListOperands(&cfg)
		_, err := lister.Run(context.TODO(), "etcd")
		Expect(err.Error()).To(ContainSubstring("could not get etcd-namespace/etcdoperator.v0.9.4-clusterwide CSV on cluster"))
	})

	It("should fail if the CSV has no owned CRDs", func() {
		csv.Spec.CustomResourceDefinitions.Owned = nil
		Expect(cfg.Client.Update(context.TODO(), csv)).To(Succeed())

		lister := action.NewOperatorListOperands(&cfg)
		_, err := lister.Run(context.TODO(), "etcd")
		Expect(err.Error()).To(ContainSubstring("no owned CustomResourceDefinitions specified on CSV etcd-namespace/etcdoperator.v0.9.4-clusterwide"))
	})

	It("should fail if the CSV is not in phase Succeeded", func() {
		csv.Status.Phase = v1alpha1.CSVPhaseFailed
		Expect(cfg.Client.Update(context.TODO(), csv)).To(Succeed())

		lister := action.NewOperatorListOperands(&cfg)
		_, err := lister.Run(context.TODO(), "etcd")
		Expect(err.Error()).To(ContainSubstring("CSV underlying operator is not in a succeeded state"))
		_, ok := err.(action.OperandListError)
		Expect(ok).To(BeTrue())
	})

	It("should fail if there is not exactly 1 operator group", func() {
		Expect(cfg.Client.Delete(context.TODO(), og)).To(Succeed())

		lister := action.NewOperatorListOperands(&cfg)
		_, err := lister.Run(context.TODO(), "etcd")
		Expect(err.Error()).To(ContainSubstring("unexpected number (0) of operator groups found in namespace etcd"))
	})

	It("should fail if an owned CRD does not exist", func() {
		Expect(cfg.Client.Delete(context.TODO(), crd)).To(Succeed())

		lister := action.NewOperatorListOperands(&cfg)
		_, err := lister.Run(context.TODO(), "etcd")
		Expect(err.Error()).To(ContainSubstring("customresourcedefinitions.apiextensions.k8s.io \"etcdclusters.etcd.database.coreos.com\" not found"))
	})

	It("should return zero operands when none exist", func() {
		Expect(cfg.Client.Delete(context.TODO(), etcdcluster1)).To(Succeed())
		Expect(cfg.Client.Delete(context.TODO(), etcdcluster2)).To(Succeed())
		Expect(cfg.Client.Delete(context.TODO(), etcdcluster3)).To(Succeed())

		lister := action.NewOperatorListOperands(&cfg)
		operands, err := lister.Run(context.TODO(), "etcd")
		Expect(err).To(BeNil())
		Expect(operands.Items).To(HaveLen(0))
	})

	It("should return operands from all namespaces", func() {
		lister := action.NewOperatorListOperands(&cfg)
		operands, err := lister.Run(context.TODO(), "etcd")
		Expect(err).To(BeNil())
		Expect(getObjectNames(*operands)).To(ConsistOf(
			types.NamespacedName{Name: "cluster1", Namespace: "ns1"},
			types.NamespacedName{Name: "cluster2", Namespace: "ns2"},
			types.NamespacedName{Name: "cluster3", Namespace: ""},
		))
	})

	It("should return operands from scoped namespaces", func() {
		og.Status.Namespaces = []string{"ns1", "ns2"}
		Expect(cfg.Client.Update(context.TODO(), og)).To(Succeed())

		lister := action.NewOperatorListOperands(&cfg)
		operands, err := lister.Run(context.TODO(), "etcd")
		Expect(err).To(BeNil())
		Expect(getObjectNames(*operands)).To(ConsistOf(
			types.NamespacedName{Name: "cluster1", Namespace: "ns1"},
			types.NamespacedName{Name: "cluster2", Namespace: "ns2"},
			types.NamespacedName{Name: "cluster3", Namespace: ""},
		))
	})

	It("should return operands from scoped namespace ns1", func() {
		og.Status.Namespaces = []string{"ns1"}
		Expect(cfg.Client.Update(context.TODO(), og)).To(Succeed())

		lister := action.NewOperatorListOperands(&cfg)
		operands, err := lister.Run(context.TODO(), "etcd")
		Expect(err).To(BeNil())
		Expect(getObjectNames(*operands)).To(ConsistOf(
			types.NamespacedName{Name: "cluster1", Namespace: "ns1"},
			types.NamespacedName{Name: "cluster3", Namespace: ""},
		))
	})

	It("should return operands from scoped namespace ns2", func() {
		og.Status.Namespaces = []string{"ns2"}
		Expect(cfg.Client.Update(context.TODO(), og)).To(Succeed())

		lister := action.NewOperatorListOperands(&cfg)
		operands, err := lister.Run(context.TODO(), "etcd")
		Expect(err).To(BeNil())
		Expect(getObjectNames(*operands)).To(ConsistOf(
			types.NamespacedName{Name: "cluster2", Namespace: "ns2"},
			types.NamespacedName{Name: "cluster3", Namespace: ""},
		))
	})

	It("should return cluster-scoped operands regardless of operator groups targetnamespaces", func() {
		og.Status.Namespaces = []string{"other"}
		Expect(cfg.Client.Update(context.TODO(), og)).To(Succeed())

		lister := action.NewOperatorListOperands(&cfg)
		operands, err := lister.Run(context.TODO(), "etcd")
		Expect(err).To(BeNil())
		Expect(getObjectNames(*operands)).To(ConsistOf(
			types.NamespacedName{Name: "cluster3", Namespace: ""},
		))
	})
})

func getObjectNames(objects unstructured.UnstructuredList) []types.NamespacedName {
	out := []types.NamespacedName{}
	for _, u := range objects.Items {
		out = append(out, types.NamespacedName{Name: u.GetName(), Namespace: u.GetNamespace()})
	}
	return out
}
