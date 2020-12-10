package fuseonopenshift

import (
	"context"
	"errors"
	"fmt"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	"net/http/httptest"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"strings"
	"testing"

	"github.com/integr8ly/integreatly-operator/pkg/apis"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"

	imagev1 "github.com/openshift/api/image/v1"
	samplesv1 "github.com/openshift/cluster-samples-operator/pkg/apis/samples/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	OperatorNamespace = "integreatly-operator"
)

type FuseOnOpenShiftScenario struct {
	Name           string
	ExpectError    bool
	ExpectedError  string
	ExpectedStatus integreatlyv1alpha1.StatusPhase
	FakeConfig     *config.ConfigReadWriterMock
	FakeClient     k8sclient.Client
	FakeMPM        *marketplace.MarketplaceInterfaceMock
	Installation   *integreatlyv1alpha1.RHMI
	Product        *integreatlyv1alpha1.RHMIProductStatus
	Recorder       record.EventRecorder
	Client         *http.Client
	Url            string
}

func getFakeConfig() *config.ConfigReadWriterMock {
	return &config.ConfigReadWriterMock{
		GetOperatorNamespaceFunc: func() string {
			return OperatorNamespace
		},
		ReadFuseOnOpenshiftFunc: func() (ready *config.FuseOnOpenshift, e error) {
			return config.NewFuseOnOpenshift(config.ProductConfig{}), nil
		},
	}
}

func getFakeServer(t *testing.T) *httptest.Server {

	var arr []byte
	var err error

	// Start a local HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.URL.String(), "fis-image-streams") {
			arr, _ = ioutil.ReadFile("./testtemplates/fis-image-streams.json")
		} else {
			arr, _ = ioutil.ReadFile("./testtemplates/test_template.json")
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(200)
		_, err = rw.Write(arr)
		if err != nil {
			t.Fatal("Failed to write file")
		}
	}))

	return server
}

func setupRecorder() record.EventRecorder {
	return record.NewFakeRecorder(50)
}

func TestFuseOnOpenShift(t *testing.T) {
	// Initialize scheme so that types required by the scenarios are available
	scheme := scheme.Scheme
	if err := apis.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to initialize scheme: %s", err)
	}

	sampleClusterConfig := &samplesv1.Config{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Spec: samplesv1.ConfigSpec{
			SkippedImagestreams: []string{"fis-java-openshift"},
			SkippedTemplates:    []string{"fuse74-console-cluster"},
		},
	}

	//Sample imagestream that's managed by the sample cluster operator
	sampleClusterImgStream := &imagev1.ImageStream{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ImageStream",
			APIVersion: "image.openshift.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fis-java-openshift",
			Namespace: fuseOnOpenshiftNs,
			Labels: map[string]string{
				"samples.operator.openshift.io/managed": "true",
			},
		},
	}

	//Sample imagestream that's created by integreatly
	integreatlyImgStream := &imagev1.ImageStream{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ImageStream",
			APIVersion: "image.openshift.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fis-java-openshift",
			Namespace: fuseOnOpenshiftNs,
			Labels: map[string]string{
				"integreatly": "true",
			},
		},
	}

	server := getFakeServer(t)

	cases := []FuseOnOpenShiftScenario{
		{
			Name:           "test error on failed config read",
			ExpectError:    true,
			ExpectedError:  fmt.Sprintf("could not retrieve %[1]s config: could not read %[1]s config", integreatlyv1alpha1.ProductFuseOnOpenshift),
			ExpectedStatus: integreatlyv1alpha1.PhaseFailed,
			Installation:   &integreatlyv1alpha1.RHMI{},
			FakeClient:     fakeclient.NewFakeClient(),
			FakeConfig: &config.ConfigReadWriterMock{
				ReadFuseOnOpenshiftFunc: func() (ready *config.FuseOnOpenshift, e error) {
					return nil, fmt.Errorf("could not read %s config", integreatlyv1alpha1.ProductFuseOnOpenshift)
				},
			},
			Product:  &integreatlyv1alpha1.RHMIProductStatus{},
			Recorder: setupRecorder(),
			Client:   server.Client(),
			Url:      server.URL + "/",
		},
		{
			Name:           "test error on invalid image stream file content",
			ExpectError:    true,
			ExpectedStatus: integreatlyv1alpha1.PhaseFailed,
			Installation:   &integreatlyv1alpha1.RHMI{},
			FakeClient: &moqclient.SigsClientInterfaceMock{
				GetFunc: func(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
					return errors.New("dummy get error")
				},
			},
			FakeConfig: getFakeConfig(),
			Product:    &integreatlyv1alpha1.RHMIProductStatus{},
			Recorder:   setupRecorder(),
			Client:     server.Client(),
			Url:        server.URL + "/",
		},
		{
			Name:           "test error on invalid image stream",
			ExpectError:    true,
			ExpectedStatus: integreatlyv1alpha1.PhaseFailed,
			Installation:   &integreatlyv1alpha1.RHMI{},
			FakeClient: &moqclient.SigsClientInterfaceMock{
				GetFunc: func(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
					return errors.New("dummy get error")
				},
			},
			FakeConfig: getFakeConfig(),
			Product:    &integreatlyv1alpha1.RHMIProductStatus{},
			Recorder:   setupRecorder(),
			Client:     server.Client(),
			Url:        server.URL + "/",
		},
		{
			Name:           "test pass on invalid template file content set to required state",
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			Installation:   &integreatlyv1alpha1.RHMI{},
			FakeClient: fakeclient.NewFakeClient(&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      templatesConfigMapName,
					Namespace: OperatorNamespace,
				},
				Data: map[string]string{
					"fis-image-streams.json":        `{ "items": [] }`,
					"fuse-console-cluster-os4.json": "invalid-file-content",
				},
			}),
			FakeConfig: getFakeConfig(),
			Product:    &integreatlyv1alpha1.RHMIProductStatus{},
			Recorder:   setupRecorder(),
			Client:     server.Client(),
			Url:        server.URL + "/",
		},
		{
			Name:           "test pass on invalid template object set to required state",
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			Installation:   &integreatlyv1alpha1.RHMI{},
			FakeClient: fakeclient.NewFakeClient(&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      templatesConfigMapName,
					Namespace: OperatorNamespace,
				},
				Data: map[string]string{
					"fis-image-streams.json": `{ "items": [] }`,
					"fuse-console-cluster-os4.json": `{
		             "name": "invalid-template"
				}`,
				},
			}),
			FakeConfig: getFakeConfig(),
			Product:    &integreatlyv1alpha1.RHMIProductStatus{},
			Recorder:   setupRecorder(),
			Client:     server.Client(),
			Url:        server.URL + "/",
		},
		{
			Name:           "test successful reconcile when resource already exists",
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			Installation:   &integreatlyv1alpha1.RHMI{},
			FakeClient:     fakeclient.NewFakeClient(integreatlyImgStream),
			FakeConfig:     getFakeConfig(),
			Product:        &integreatlyv1alpha1.RHMIProductStatus{},
			Recorder:       setupRecorder(),
			Client:         server.Client(),
			Url:            server.URL + "/",
		},
		{
			Name:           "test successful reconcile without sample cluster operator installed",
			ExpectError:    false,
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			Installation:   &integreatlyv1alpha1.RHMI{},
			FakeClient:     fakeclient.NewFakeClient(),
			FakeConfig:     getFakeConfig(),
			Product:        &integreatlyv1alpha1.RHMIProductStatus{},
			Recorder:       setupRecorder(),
			Client:         server.Client(),
			Url:            server.URL + "/",
		},
		{
			Name:           "test successful reconcile with sample cluster operator installed",
			ExpectError:    false,
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			Installation:   &integreatlyv1alpha1.RHMI{},
			FakeClient:     fakeclient.NewFakeClient(sampleClusterConfig, sampleClusterImgStream),
			FakeConfig:     getFakeConfig(),
			Product:        &integreatlyv1alpha1.RHMIProductStatus{},
			Recorder:       setupRecorder(),
			Client:         server.Client(),
			Url:            server.URL + "/",
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			testReconciler, err := NewReconciler(
				tc.FakeConfig,
				tc.Installation,
				tc.FakeMPM,
				tc.Recorder,
				tc.Client,
				tc.Url,
				getLogger(),
			)

			if err != nil && err.Error() != tc.ExpectedError {
				t.Fatalf("unexpected error : '%v', expected: '%v'", err, tc.ExpectedError)
			}

			if err == nil && tc.ExpectedError != "" {
				t.Fatalf("expected error '%v' and got nil", tc.ExpectedError)
			}

			// if we expect errors creating the reconciler, don't try to use it
			if tc.ExpectedError != "" {
				return
			}

			status, err := testReconciler.Reconcile(context.TODO(), tc.Installation, tc.Product, tc.FakeClient)
			if err != nil && !tc.ExpectError {
				t.Fatalf("expected error but got one: %v", err)
			}

			if err == nil && tc.ExpectError {
				t.Fatal("expected error but got none")
			}

			if status != tc.ExpectedStatus {
				t.Fatalf("Expected status: '%v', got: '%v'", tc.ExpectedStatus, status)
			}
		})
	}
}

func getLogger() l.Logger {
	return l.NewLoggerWithContext(l.Fields{l.ProductLogContext: integreatlyv1alpha1.ProductApicurioRegistry})
}
