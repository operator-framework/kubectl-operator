package action_test

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	internalaction "github.com/operator-framework/kubectl-operator/internal/pkg/v1/action"
)

type mockCreateClient struct {
	*mockCreator
	*mockGetter
	*mockDeleter
	createCatalog *olmv1.ClusterCatalog
}

func (mcc *mockCreateClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	mcc.createCatalog = obj.(*olmv1.ClusterCatalog)
	return mcc.mockCreator.Create(ctx, obj, opts...)
}

var _ = Describe("CatalogCreate", func() {
	pollInterval := 20
	expectedCatalog := olmv1.ClusterCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "testcatalog",
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
		mockClient := &mockCreateClient{&mockCreator{createErr: expectedErr}, nil, nil, &expectedCatalog}

		creator := internalaction.NewCatalogCreate(mockClient)
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

		// there is no way of testing a happy path in unit tests because we have no way to
		// set/mock the catalog status condition we're waiting for in waitUntilCatalogStatusCondition
		// but we can still at least verify that CR would have been created with expected attribute values
		validateCreateCatalog(mockClient.createCatalog, &expectedCatalog)
	})

	It("fails waiting for created catalog status, successfully cleans up", func() {
		expectedErr := errors.New("get failed")
		mockClient := &mockCreateClient{&mockCreator{}, &mockGetter{getErr: expectedErr}, &mockDeleter{}, nil}

		creator := internalaction.NewCatalogCreate(mockClient)
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
		mockClient := &mockCreateClient{&mockCreator{}, &mockGetter{getErr: getErr}, &mockDeleter{deleteErr: deleteErr}, nil}

		creator := internalaction.NewCatalogCreate(mockClient)
		err := creator.Run(context.TODO())

		Expect(err).NotTo(BeNil())
		Expect(err).To(MatchError(getErr))
		Expect(mockClient.createCalled).To(Equal(1))
		Expect(mockClient.getCalled).To(Equal(1))
		Expect(mockClient.deleteCalled).To(Equal(1))
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
