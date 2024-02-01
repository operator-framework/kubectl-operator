package fetcher

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/operator-framework/catalogd/api/core/v1alpha1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestFetcher(t *testing.T) {
	var tests = []struct {
		name             string
		fetcher          *Fetcher
		filters          []CatalogFilterFunc
		expectedCatalogs []v1alpha1.Catalog
	}{
		{
			name: "no catalogs exist, no catalogs returned",
			fetcher: func() *Fetcher {
				scheme := runtime.NewScheme()
				err := v1alpha1.AddToScheme(scheme)
				require.NoError(t, err)

				return New(fake.NewClientBuilder().WithScheme(scheme).Build())
			}(),
			expectedCatalogs: []v1alpha1.Catalog{},
		},
		{
			name: "catalogs exist, no filters, all catalogs returned",
			fetcher: func() *Fetcher {
				scheme := runtime.NewScheme()
				err := v1alpha1.AddToScheme(scheme)
				require.NoError(t, err)

				objs := []client.Object{
					&v1alpha1.Catalog{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-catalog",
						},
					},
				}
				return New(fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build())
			}(),
			expectedCatalogs: []v1alpha1.Catalog{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-catalog",
					},
				},
			},
		},
		{
			name: "catalogs exist, name filter, only matching catalogs returned",
			fetcher: func() *Fetcher {
				scheme := runtime.NewScheme()
				err := v1alpha1.AddToScheme(scheme)
				require.NoError(t, err)

				objs := []client.Object{
					&v1alpha1.Catalog{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-catalog",
						},
					}, &v1alpha1.Catalog{
						ObjectMeta: metav1.ObjectMeta{
							Name: "another-catalog",
						},
					},
				}
				return New(fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build())
			}(),
			expectedCatalogs: []v1alpha1.Catalog{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-catalog",
					},
				},
			},
			filters: []CatalogFilterFunc{
				WithNameFilter("test-catalog"),
			},
		},
		{
			name: "catalogs exist, unpacked filter, only matching catalogs returned",
			fetcher: func() *Fetcher {
				scheme := runtime.NewScheme()
				err := v1alpha1.AddToScheme(scheme)
				require.NoError(t, err)

				objs := []client.Object{
					&v1alpha1.Catalog{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-catalog",
						},
						Status: v1alpha1.CatalogStatus{
							Conditions: []metav1.Condition{
								{
									Type:   v1alpha1.TypeUnpacked,
									Status: metav1.ConditionTrue,
								},
							},
						},
					}, &v1alpha1.Catalog{
						ObjectMeta: metav1.ObjectMeta{
							Name: "another-catalog",
						},
					},
				}
				return New(fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build())
			}(),
			expectedCatalogs: []v1alpha1.Catalog{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-catalog",
					},
					Status: v1alpha1.CatalogStatus{
						Conditions: []metav1.Condition{
							{
								Type:   v1alpha1.TypeUnpacked,
								Status: metav1.ConditionTrue,
							},
						},
					},
				},
			},
			filters: []CatalogFilterFunc{
				WithUnpackedFilter(),
			},
		},
	}

	for _, tt := range tests {
		fetch := tt.fetcher
		filters := tt.filters
		expectedCatalogs := tt.expectedCatalogs
		t.Run(tt.name, func(t *testing.T) {
			catalogs, err := fetch.FetchCatalogs(context.Background(), filters...)
			require.NoError(t, err)
			diff := cmp.Diff(expectedCatalogs, catalogs, cmpopts.IgnoreFields(metav1.ObjectMeta{}, "ResourceVersion"))
			require.Equal(t, diff, "")
		})
	}
}
