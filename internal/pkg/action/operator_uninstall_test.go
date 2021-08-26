package action_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "github.com/operator-framework/api/pkg/operators/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	internalaction "github.com/operator-framework/kubectl-operator/internal/pkg/action"
	"github.com/operator-framework/kubectl-operator/internal/pkg/operand"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

var _ = Describe("OperatorUninstall", func() {
	const etcd = "etcd"
	var (
		cfg          action.Configuration
		operator     *v1.Operator
		csv          *v1alpha1.ClusterServiceVersion
		crd          *apiextv1.CustomResourceDefinition
		og           *v1.OperatorGroup
		sub          *v1alpha1.Subscription
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

		sub = &v1alpha1.Subscription{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "etcd-sub",
				Namespace: "etcd-namespace",
			},
			Spec: &v1alpha1.SubscriptionSpec{
				Package: "etcd",
			},
			Status: v1alpha1.SubscriptionStatus{
				InstalledCSV: "etcdoperator.v0.9.4-clusterwide",
			},
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
			WithObjects(operator, csv, og, sub, crd, etcdcluster1, etcdcluster2, etcdcluster3).
			WithScheme(sch).
			Build()
		cfg.Scheme = sch
		cfg.Client = cl
		cfg.Namespace = "etcd-namespace"
	})

	It("should fail due to missing subscription", func() {
		uninstaller := internalaction.NewOperatorUninstall(&cfg)
		// switch to package without a subscription for it
		uninstaller.Package = "redis"
		err := uninstaller.Run(context.TODO())
		Expect(err).To(MatchError(&internalaction.ErrPackageNotFound{PackageName: "redis"}))
	})

	It("should not fail due to missing csv", func() {
		// switch to missing csv
		// this is not an error condition, we simply delete the subscription and exit
		sub.Status.InstalledCSV = ""
		Expect(cfg.Client.Update(context.TODO(), sub)).To(Succeed())

		uninstaller := internalaction.NewOperatorUninstall(&cfg)
		uninstaller.Package = etcd
		uninstaller.OperandStrategy = operand.Ignore
		err := uninstaller.Run(context.TODO())
		Expect(err).To(BeNil())

		subKey := types.NamespacedName{Name: "etcd-sub", Namespace: "etcd-namespace"}
		s := &v1alpha1.Subscription{}
		Expect(cfg.Client.Get(context.TODO(), subKey, s)).To(WithTransform(apierrors.IsNotFound, BeTrue()))
	})

	It("should fail due to invalid operand deletion strategy", func() {
		uninstaller := internalaction.NewOperatorUninstall(&cfg)
		uninstaller.Package = etcd
		uninstaller.OperandStrategy = "foo"
		err := uninstaller.Run(context.TODO())
		Expect(err.Error()).To(ContainSubstring("unknown operand deletion strategy"))
	})

	It("should error with operands on cluster when default cancel strategy is set", func() {
		uninstaller := internalaction.NewOperatorUninstall(&cfg)
		uninstaller.Package = etcd
		err := uninstaller.Run(context.TODO())
		Expect(err).To(MatchError(operand.ErrCancelStrategy))
	})

	It("should ignore operands and delete sub and csv when ignore strategy is set", func() {
		uninstaller := internalaction.NewOperatorUninstall(&cfg)
		uninstaller.Package = etcd
		uninstaller.OperandStrategy = operand.Ignore
		err := uninstaller.Run(context.TODO())
		Expect(err).To(BeNil())

		subKey := types.NamespacedName{Name: "etcd-sub", Namespace: "etcd-namespace"}
		s := &v1alpha1.Subscription{}
		Expect(cfg.Client.Get(context.TODO(), subKey, s)).To(WithTransform(apierrors.IsNotFound, BeTrue()))

		csvKey := types.NamespacedName{Name: "etcdoperator.v0.9.4-clusterwide", Namespace: "etcd-namespace"}
		csv := &v1alpha1.ClusterServiceVersion{}
		Expect(cfg.Client.Get(context.TODO(), csvKey, csv)).To(WithTransform(apierrors.IsNotFound, BeTrue()))

		//check operands are still around
		etcd1Key := types.NamespacedName{Name: "cluster1", Namespace: "ns1"}
		Expect(cfg.Client.Get(context.TODO(), etcd1Key, etcdcluster1)).To(Succeed())

		etcd2Key := types.NamespacedName{Name: "cluster2", Namespace: "ns2"}
		Expect(cfg.Client.Get(context.TODO(), etcd2Key, etcdcluster2)).To(Succeed())

		etcd3Key := types.NamespacedName{Name: "cluster3"}
		Expect(cfg.Client.Get(context.TODO(), etcd3Key, etcdcluster3)).To(Succeed())
	})

	It("should delete sub, csv, and operands when delete strategy is set", func() {
		uninstaller := internalaction.NewOperatorUninstall(&cfg)
		uninstaller.Package = etcd
		uninstaller.OperandStrategy = operand.Delete
		err := uninstaller.Run(context.TODO())
		Expect(err).To(BeNil())

		subKey := types.NamespacedName{Name: "etcd-sub", Namespace: "etcd-namespace"}
		s := &v1alpha1.Subscription{}
		Expect(cfg.Client.Get(context.TODO(), subKey, s)).To(WithTransform(apierrors.IsNotFound, BeTrue()))

		csvKey := types.NamespacedName{Name: "etcdoperator.v0.9.4-clusterwide", Namespace: "etcd-namespace"}
		csv := &v1alpha1.ClusterServiceVersion{}
		Expect(cfg.Client.Get(context.TODO(), csvKey, csv)).To(WithTransform(apierrors.IsNotFound, BeTrue()))

		etcd1Key := types.NamespacedName{Name: "cluster1", Namespace: "ns1"}
		Expect(cfg.Client.Get(context.TODO(), etcd1Key, etcdcluster1)).To(WithTransform(apierrors.IsNotFound, BeTrue()))

		etcd2Key := types.NamespacedName{Name: "cluster2", Namespace: "ns2"}
		Expect(cfg.Client.Get(context.TODO(), etcd2Key, etcdcluster2)).To(WithTransform(apierrors.IsNotFound, BeTrue()))

		etcd3Key := types.NamespacedName{Name: "cluster3"}
		Expect(cfg.Client.Get(context.TODO(), etcd3Key, etcdcluster3)).To(WithTransform(apierrors.IsNotFound, BeTrue()))
	})
	It("should delete sub and operatorgroup when no CSV is found", func() {
		uninstaller := internalaction.NewOperatorUninstall(&cfg)
		uninstaller.Package = etcd
		uninstaller.OperandStrategy = operand.Ignore
		uninstaller.DeleteOperatorGroups = true

		sub.Status.InstalledCSV = "foo" // returns nil CSV
		Expect(cfg.Client.Update(context.TODO(), sub)).To(Succeed())

		err := uninstaller.Run(context.TODO())
		Expect(err).To(BeNil())

		subKey := types.NamespacedName{Name: "etcd-sub", Namespace: "etcd-namespace"}
		s := &v1alpha1.Subscription{}
		Expect(cfg.Client.Get(context.TODO(), subKey, s)).To(WithTransform(apierrors.IsNotFound, BeTrue()))

		ogKey := types.NamespacedName{Name: "etcd", Namespace: "etcd-namespace"}
		og := &v1.OperatorGroup{}
		Expect(cfg.Client.Get(context.TODO(), ogKey, og)).To(WithTransform(apierrors.IsNotFound, BeTrue()))
	})
})
