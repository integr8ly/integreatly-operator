package products

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/integr8ly/integreatly-operator/api/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/products/cloudresources"
	"github.com/integr8ly/integreatly-operator/pkg/products/grafana"
	"github.com/integr8ly/integreatly-operator/pkg/products/marin3r"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	"github.com/integr8ly/integreatly-operator/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"net/http"
	"net/http/httptest"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"testing"
)

var (
	defaultTestNamespace = "redhat-rhoam-test"
)

func createTestLogger(productName string) l.Logger {
	return l.NewLoggerWithContext(l.Fields{l.ProductLogContext: productName})
}

func configureTestServer(t *testing.T) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		list := &metav1.APIResourceList{
			// "integreatly.org/v1alpha1"
			GroupVersion: "integreatly.org/v1alpha1",
			APIResources: []metav1.APIResource{},
		}

		output, err := json.Marshal(list)
		if err != nil {
			t.Errorf("unexpected encoding error: %v", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(output)
		if err != nil {
			t.Fatal(err)
		}
	}))
	return server
}

func TestNewReconciler(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	server := configureTestServer(t)
	defer server.Close()

	restConfig := &rest.Config{Host: server.URL}

	mgr, err := ctrl.NewManager(restConfig, ctrl.Options{
		Scheme: scheme,
	})
	if err != nil {
		t.Fatal(err)
	}

	installation := &integreatlyv1alpha1.RHMI{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RHMI",
			APIVersion: integreatlyv1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: defaultTestNamespace,
		},
	}

	productsInstallationLoader := marketplace.NewFSProductInstallationLoader(
		fmt.Sprintf("../../%s", marketplace.GetProductsInstallationPath()),
	)

	type args struct {
		product                    v1alpha1.ProductName
		rc                         *rest.Config
		configManager              config.ConfigReadWriter
		installation               *integreatlyv1alpha1.RHMI
		mgr                        manager.Manager
		logger                     l.Logger
		productsInstallationLoader marketplace.ProductsInstallationLoader
	}
	tests := []struct {
		name             string
		args             args
		wantedReconciler Interface
		wantErr          bool
	}{
		{
			name: "Test can handle unknown product",
			args: args{
				product:                    "bad product",
				rc:                         restConfig,
				configManager:              &config.ConfigReadWriterMock{},
				installation:               installation,
				mgr:                        mgr,
				logger:                     createTestLogger("bad-product"),
				productsInstallationLoader: productsInstallationLoader,
			},
			wantedReconciler: &NoOp{},
			wantErr:          true,
		},
		{
			name: "Fail on bad RHSSO Config",
			args: args{
				product: integreatlyv1alpha1.ProductRHSSO,
				rc:      restConfig,
				configManager: &config.ConfigReadWriterMock{
					ReadRHSSOFunc: func() (ready *config.RHSSO, e error) {
						return nil, errors.New("could not read rhsso config")
					},
				},
				installation:               installation,
				mgr:                        mgr,
				logger:                     createTestLogger(string(integreatlyv1alpha1.ProductRHSSO)),
				productsInstallationLoader: productsInstallationLoader,
			},
			wantedReconciler: nil,
			wantErr:          true,
		},
		{
			name: "Fail on bad ProductRHSSOUser Config",
			args: args{
				product: integreatlyv1alpha1.ProductRHSSOUser,
				rc:      restConfig,
				configManager: &config.ConfigReadWriterMock{
					ReadRHSSOUserFunc: func() (ready *config.RHSSOUser, e error) {
						return nil, errors.New("could not read rhsso user config")
					},
				},
				installation:               installation,
				mgr:                        mgr,
				logger:                     createTestLogger(string(integreatlyv1alpha1.ProductRHSSOUser)),
				productsInstallationLoader: productsInstallationLoader,
			},
			wantedReconciler: nil,
			wantErr:          true,
		},
		{
			name: "Fail on bad 3scale Config",
			args: args{
				product: integreatlyv1alpha1.Product3Scale,
				rc:      restConfig,
				configManager: &config.ConfigReadWriterMock{
					ReadThreeScaleFunc: func() (ready *config.ThreeScale, e error) {
						return nil, errors.New("could not read 3scale config")
					},
				},
				installation:               installation,
				mgr:                        mgr,
				logger:                     createTestLogger(string(integreatlyv1alpha1.Product3Scale)),
				productsInstallationLoader: productsInstallationLoader,
			},
			wantedReconciler: nil,
			wantErr:          true,
		},
		{
			name: "Fail on bad ProductCloudResources Config",
			args: args{
				product: integreatlyv1alpha1.ProductCloudResources,
				rc:      restConfig,
				configManager: &config.ConfigReadWriterMock{
					ReadCloudResourcesFunc: func() (ready *config.CloudResources, e error) {
						return nil, errors.New("could not read cloud resources config")
					},
				},
				installation:               installation,
				mgr:                        mgr,
				logger:                     createTestLogger(string(integreatlyv1alpha1.ProductCloudResources)),
				productsInstallationLoader: productsInstallationLoader,
			},
			wantedReconciler: (*cloudresources.Reconciler)(nil),
			wantErr:          true,
		},
		{
			name: "Fail on bad ProductMarin3r Config",
			args: args{
				product: integreatlyv1alpha1.ProductMarin3r,
				rc:      restConfig,
				configManager: &config.ConfigReadWriterMock{
					ReadMarin3rFunc: func() (ready *config.Marin3r, e error) {
						return nil, errors.New("could not read marin3r config")
					},
				},
				installation:               installation,
				mgr:                        mgr,
				logger:                     createTestLogger(string(integreatlyv1alpha1.ProductMarin3r)),
				productsInstallationLoader: productsInstallationLoader,
			},
			wantedReconciler: (*marin3r.Reconciler)(nil),
			wantErr:          true,
		},
		{
			name: "Fail on bad ProductGrafana Config",
			args: args{
				product: integreatlyv1alpha1.ProductGrafana,
				rc:      restConfig,
				configManager: &config.ConfigReadWriterMock{
					ReadGrafanaFunc: func() (ready *config.Grafana, e error) {
						return nil, errors.New("could not read grafana config")
					},
				},
				installation:               installation,
				mgr:                        mgr,
				logger:                     createTestLogger(string(integreatlyv1alpha1.ProductGrafana)),
				productsInstallationLoader: productsInstallationLoader,
			},
			wantedReconciler: (*grafana.Reconciler)(nil),
			wantErr:          true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotReconciler, err := NewReconciler(
				tt.args.product,
				tt.args.rc,
				tt.args.configManager,
				tt.args.installation,
				tt.args.mgr,
				tt.args.logger,
				tt.args.productsInstallationLoader,
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewReconciler() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotReconciler, tt.wantedReconciler) {
				t.Errorf("NewReconciler() gotReconciler = %v, want %v", gotReconciler, tt.wantedReconciler)
			}
		})
	}
}
