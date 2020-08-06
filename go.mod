module github.com/operator-framework/kubectl-operator

go 1.13

require (
	github.com/operator-framework/api v0.3.7
	github.com/operator-framework/operator-lifecycle-manager v0.0.0-20200521062108-408ca95d458f
	github.com/operator-framework/operator-registry v1.12.5
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.18.6
	k8s.io/apiextensions-apiserver v0.18.6
	k8s.io/apimachinery v0.18.6
	k8s.io/client-go v0.18.6
	sigs.k8s.io/controller-runtime v0.6.2
	sigs.k8s.io/yaml v1.2.0
)
