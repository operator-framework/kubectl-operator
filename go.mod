module github.com/operator-framework/kubectl-operator

go 1.13

require (
	github.com/containerd/containerd v1.4.3
	github.com/opencontainers/image-spec v1.0.2-0.20190823105129-775207bd45b6
	github.com/operator-framework/api v0.3.7
	github.com/operator-framework/operator-lifecycle-manager v0.0.0-20200521062108-408ca95d458f
	github.com/operator-framework/operator-registry v1.12.5
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.20.2
	k8s.io/apiextensions-apiserver v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	sigs.k8s.io/controller-runtime v0.8.0
)
