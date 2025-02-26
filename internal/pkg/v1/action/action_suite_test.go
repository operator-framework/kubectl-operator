package action_test

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1catalogd "github.com/operator-framework/catalogd/api/v1"
)

func TestCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Internal v1 action Suite")
}

func setupTestCatalogs(n int) []client.Object {
	var result []client.Object
	for i := 1; i <= n; i++ {
		result = append(result, newClusterCatalog(fmt.Sprintf("cat%d", i)))
	}

	return result
}

func newClusterCatalog(name string) *olmv1catalogd.ClusterCatalog {
	return &olmv1catalogd.ClusterCatalog{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
}
