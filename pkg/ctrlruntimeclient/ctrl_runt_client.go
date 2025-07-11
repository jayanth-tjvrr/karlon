package ctrlruntimeclient

import (
	appset "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	kkarlonv1 "github.com/jayanth-tjvrr/karlon/pkg/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(kkarlonv1.AddToScheme(scheme))
	utilruntime.Must(appset.AddToScheme(scheme))
}

func NewClient(config *rest.Config) (client.Client, error) {
	return client.New(config, client.Options{
		Scheme: scheme,
	})
}
