package streamer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/operator-framework/catalogd/api/core/v1alpha1"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	cgotesting "k8s.io/client-go/testing"
)

type MockResponseWrapper struct {
	shouldError bool
	content     []byte
}

func (m *MockResponseWrapper) Stream(_ context.Context) (io.ReadCloser, error) {
	if m.shouldError {
		return nil, fmt.Errorf("error")
	}
	return io.NopCloser(bytes.NewReader(m.content)), nil
}

func (m *MockResponseWrapper) DoRaw(_ context.Context) ([]byte, error) {
	if m.shouldError {
		return nil, fmt.Errorf("error")
	}
	return m.content, nil
}

func TestStreamer(t *testing.T) {
	var tests = []struct {
		name            string
		streamer        *Streamer
		catalog         v1alpha1.Catalog
		expectedContent string
		expectError     bool
	}{
		{
			name: "catalog is unpacked and has content, content is returned",
			streamer: func() *Streamer {
				kc := fake.NewSimpleClientset()
				kc.ProxyReactionChain = []cgotesting.ProxyReactor{
					&cgotesting.SimpleProxyReactor{
						Resource: "services",
						Reaction: func(action cgotesting.Action) (handled bool, ret rest.ResponseWrapper, err error) {
							return true, &MockResponseWrapper{content: []byte("test")}, nil
						},
					},
				}
				return New(kc.CoreV1())
			}(),
			catalog: v1alpha1.Catalog{
				Status: v1alpha1.CatalogStatus{
					Conditions: []v1.Condition{
						{
							Type:   v1alpha1.TypeUnpacked,
							Status: v1.ConditionTrue,
						},
					},
					ContentURL: "http://test-catalog.test-namespace.svc/catalogs/test-catalog/all.json",
				},
			},
			expectedContent: "test",
		},
		{
			name: "catalog is not unpacked, error is returned",
			streamer: func() *Streamer {
				kc := fake.NewSimpleClientset()
				return New(kc.CoreV1())
			}(),
			catalog:     v1alpha1.Catalog{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		stream := tt.streamer
		catalog := tt.catalog
		expectError := tt.expectError
		expectedContent := tt.expectedContent
		t.Run(tt.name, func(t *testing.T) {
			rc, err := stream.StreamCatalogContents(context.Background(), catalog)
			if expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			t.Cleanup(func() {
				rc.Close()
			})
			content, err := io.ReadAll(rc)
			require.NoError(t, err)
			require.Equal(t, expectedContent, string(content))
		})
	}
}
