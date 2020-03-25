package functional

import (
	"fmt"
	"testing"

	"github.com/integr8ly/integreatly-operator/pkg/apis"
	"github.com/integr8ly/integreatly-operator/test/common"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	extscheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	"k8s.io/apimachinery/pkg/runtime"
	cached "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/kubernetes"
	cgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"
	config2 "sigs.k8s.io/controller-runtime/pkg/client/config"
)

func TestIntegreatly(t *testing.T) {
	config, err := config2.GetConfig()
	if err != nil {
		t.Fatal(err)
	}
	testingContext, err := newTestingContext(config)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("Integreatly Happy Path Tests", func(t *testing.T) {
		for _, test := range common.ALL_TESTS {
			t.Run(test.Description, func(t *testing.T) {
				test.Test(t, testingContext)
			})
		}
		for _, test := range common.AFTER_INSTALL_TESTS {
			t.Run(test.Description, func(t *testing.T) {
				test.Test(t, testingContext)
			})
		}
	})
}

func newTestingContext(kubeConfig *rest.Config) (*common.TestingContext, error) {
	kubeclient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build the kubeclient: %v", err)
	}

	apiextensions, err := clientset.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build the apiextension client: %v", err)
	}

	scheme := runtime.NewScheme()
	if err := cgoscheme.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add cgo scheme to runtime scheme: (%v)", err)
	}
	if err := extscheme.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add api extensions scheme to runtime scheme: (%v)", err)
	}
	if err := apis.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add integreatly scheme to runtime scheme: (%v)", err)
	}

	cachedDiscoveryClient := cached.NewMemCacheClient(kubeclient.Discovery())
	restMapper := restmapper.NewDeferredDiscoveryRESTMapper(cachedDiscoveryClient)
	restMapper.Reset()

	dynClient, err := dynclient.New(kubeConfig, dynclient.Options{Scheme: scheme, Mapper: restMapper})
	if err != nil {
		return nil, fmt.Errorf("failed to build the dynamic client: %v", err)
	}
	return &common.TestingContext{
		Client:          dynClient,
		KubeConfig:      kubeConfig,
		KubeClient:      kubeclient,
		ExtensionClient: apiextensions,
	}, nil
}
