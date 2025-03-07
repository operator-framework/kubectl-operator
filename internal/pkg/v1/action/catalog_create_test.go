package action_test

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/kubectl-operator/pkg/action"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	internalaction "github.com/operator-framework/kubectl-operator/internal/pkg/v1/action"
)

var _ = Describe("CatalogCreate", func() {
	catalogName := "testcatalog"
	pollInterval := 20
	expectedCatalog := olmv1.ClusterCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:   catalogName,
			Labels: map[string]string{"a": "b"},
		},
		Spec: olmv1.ClusterCatalogSpec{
			Source: olmv1.CatalogSource{
				Type: olmv1.SourceTypeImage,
				Image: &olmv1.ImageSource{
					Ref:                 "testcatalog:latest",
					PollIntervalMinutes: &pollInterval,
				},
			},
			Priority:         77,
			AvailabilityMode: olmv1.AvailabilityModeAvailable,
		},
	}

	It("fails creating catalog", func() {
		expectedErr := errors.New("create failed")
		mockClient := fakeClient{createErr: expectedErr}
		mockClient.Initialize()

		creator := internalaction.NewCatalogCreate(&action.Configuration{Client: mockClient})
		creator.Available = true
		creator.CatalogName = expectedCatalog.Name
		creator.ImageSourceRef = expectedCatalog.Spec.Source.Image.Ref
		creator.Priority = expectedCatalog.Spec.Priority
		creator.Labels = expectedCatalog.Labels
		creator.PollIntervalMinutes = *expectedCatalog.Spec.Source.Image.PollIntervalMinutes
		err := creator.Run(context.TODO())

		Expect(err).NotTo(BeNil())
		Expect(err).To(MatchError(expectedErr))
		Expect(mockClient.createCalled).To(Equal(1))
	})

	It("fails waiting for created catalog status, successfully cleans up", func() {
		expectedErr := errors.New("get failed")
		mockClient := fakeClient{getErr: expectedErr}
		Expect(mockClient.Initialize()).To(Succeed())

		creator := internalaction.NewCatalogCreate(&action.Configuration{Client: mockClient})
		creator.CatalogName = expectedCatalog.Name
		err := creator.Run(context.TODO())

		Expect(err).NotTo(BeNil())
		Expect(err).To(MatchError(expectedErr))
		Expect(mockClient.createCalled).To(Equal(1))
		Expect(mockClient.getCalled).To(Equal(1))
		Expect(mockClient.deleteCalled).To(Equal(1))
	})

	It("fails waiting for created catalog status, fails clean up", func() {
		getErr := errors.New("get failed")
		deleteErr := errors.New("delete failed")
		mockClient := fakeClient{deleteErr: deleteErr, getErr: getErr}
		Expect(mockClient.Initialize()).To(Succeed())

		creator := internalaction.NewCatalogCreate(&action.Configuration{Client: mockClient})
		// fakeClient requires at least the catalogName to be set to run
		creator.CatalogName = expectedCatalog.Name
		err := creator.Run(context.TODO())

		Expect(err).NotTo(BeNil())
		Expect(err).To(MatchError(getErr))
		Expect(mockClient.createCalled).To(Equal(1))
		Expect(mockClient.getCalled).To(Equal(1))
		Expect(mockClient.deleteCalled).To(Equal(1))
	})
	It("succeeds creating catalog", func() {
		mockClient := fakeClient{
			transformers: []objectTransformer{
				{
					verb:      verbCreate,
					objectKey: types.NamespacedName{Name: catalogName},
					transformFunc: func(obj *client.Object) {
						if obj == nil {
							return
						}
						catalogObj, ok := (*obj).(*olmv1.ClusterCatalog)
						if !ok {
							return
						}
						catalogObj.Status.Conditions = []metav1.Condition{{Type: olmv1.TypeServing, Status: metav1.ConditionTrue}}
					},
				},
			},
		}
		Expect(mockClient.Initialize()).To(Succeed())

		creator := internalaction.NewCatalogCreate(&action.Configuration{Client: mockClient})
		creator.Available = true
		creator.CatalogName = expectedCatalog.Name
		creator.ImageSourceRef = expectedCatalog.Spec.Source.Image.Ref
		creator.Priority = expectedCatalog.Spec.Priority
		creator.Labels = expectedCatalog.Labels
		creator.PollIntervalMinutes = *expectedCatalog.Spec.Source.Image.PollIntervalMinutes
		Expect(creator.Run(context.TODO())).To(Succeed())

		Expect(mockClient.createCalled).To(Equal(1))

		actualCatalog := &olmv1.ClusterCatalog{TypeMeta: metav1.TypeMeta{Kind: "ClusterCatalog", APIVersion: "olm.operatorframework.io/v1"}}
		mockClient.Client.Get(context.TODO(), types.NamespacedName{Name: catalogName}, actualCatalog)
		validateCreateCatalog(actualCatalog, &expectedCatalog)
	})
})

func validateCreateCatalog(actual, expected *olmv1.ClusterCatalog) {
	Expect(actual.Spec.Source.Image.Ref).To(Equal(expected.Spec.Source.Image.Ref))
	Expect(actual.Spec.Source.Image.PollIntervalMinutes).To(Equal(expected.Spec.Source.Image.PollIntervalMinutes))
	Expect(actual.Spec.AvailabilityMode).To(Equal(expected.Spec.AvailabilityMode))
	Expect(actual.Labels).To(HaveLen(len(expected.Labels)))
	for k, v := range expected.Labels {
		Expect(actual.Labels).To(HaveKeyWithValue(k, v))
	}
	Expect(actual.Spec.Priority).To(Equal(expected.Spec.Priority))
}
