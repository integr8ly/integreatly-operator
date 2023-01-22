package threescale

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/foxcpp/go-mockdns"
	"github.com/integr8ly/integreatly-operator/pkg/products/mcg"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"
	obv1 "github.com/kube-object-storage/lib-bucket-provisioner/pkg/apis/objectbucket.io/v1alpha1"
	noobaav1 "github.com/noobaa/noobaa-operator/v5/pkg/apis/noobaa/v1alpha1"
	customdomainv1alpha1 "github.com/openshift/custom-domains-operator/api/v1alpha1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/integr8ly/integreatly-operator/test/utils"

	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	k8sTypes "k8s.io/apimachinery/pkg/types"

	threescalev1 "github.com/3scale/3scale-operator/apis/apps/v1alpha1"
	threescaleAmp "github.com/3scale/3scale-operator/pkg/3scale/amp/component"
	crov1 "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1"
	"github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1/types"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	keycloak "github.com/integr8ly/keycloak-client/apis/keycloak/v1alpha1"
	configv1 "github.com/openshift/api/config/v1"
	routev1 "github.com/openshift/api/route/v1"
	usersv1 "github.com/openshift/api/user/v1"
	appsv1Client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	fakeoauthClient "github.com/openshift/client-go/oauth/clientset/versioned/fake"
	oauthClient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"

	"github.com/integr8ly/integreatly-operator/pkg/resources/sts"
	openshiftappsv1 "github.com/openshift/api/apps/v1"
	cloudcredentialv1 "github.com/openshift/api/operator/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	integreatlyOperatorNamespace = "integreatly-operator-ns"
	localProductDeclaration      = marketplace.LocalProductDeclaration("integreatly-3scale")
)

func setupRecorder() record.EventRecorder {
	return record.NewFakeRecorder(50)
}

func getLogger() l.Logger {
	return l.NewLoggerWithContext(l.Fields{l.ProductLogContext: integreatlyv1alpha1.Product3Scale})
}

type fields struct {
	sigsClient       k8sclient.Client
	mpm              *marketplace.MarketplaceInterfaceMock
	appsv1Client     appsv1Client.AppsV1Interface
	oauthv1Client    oauthClient.OauthV1Interface
	recorder         record.EventRecorder
	threeScaleClient *ThreeScaleInterfaceMock
	fakeConfig       *config.ConfigReadWriterMock
}
type args struct {
	installation  *integreatlyv1alpha1.RHMI
	productStatus *integreatlyv1alpha1.RHMIProductStatus
	productConfig quota.ProductConfig
	uninstall     bool
}

type ThreeScaleTestScenario struct {
	name        string
	fields      fields
	args        args
	assert      bool
	want        integreatlyv1alpha1.StatusPhase
	wantErr     bool
	mockNetwork bool
}

func TestReconciler_Reconcile3scale(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	objects := getSuccessfullRHOAMTestPreReqs(integreatlyOperatorNamespace, defaultInstallationNamespace)
	objects = append(objects, getValidInstallation(integreatlyv1alpha1.InstallationTypeMultitenantManagedApi))

	tests := []ThreeScaleTestScenario{
		{
			name: "Test successful installation of MT RHOAM",
			fields: fields{
				sigsClient: moqclient.NewSigsClientMoqWithScheme(scheme, objects...),
				mpm: &marketplace.MarketplaceInterfaceMock{
					InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval, catalogSourceReconciler marketplace.CatalogSourceReconciler) error {

						return nil
					},
					GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (plans *operatorsv1alpha1.InstallPlan, subscription *operatorsv1alpha1.Subscription, e error) {
						return &operatorsv1alpha1.InstallPlan{
								ObjectMeta: metav1.ObjectMeta{
									Name: "3scale-install-plan",
								},
								Status: operatorsv1alpha1.InstallPlanStatus{
									Phase: operatorsv1alpha1.InstallPlanPhaseComplete,
								},
							}, &operatorsv1alpha1.Subscription{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "rhmi-3scale",
									Namespace: "3scale",
								},
								Status: operatorsv1alpha1.SubscriptionStatus{
									Install: &operatorsv1alpha1.InstallPlanReference{
										Name: "3scale-install-plan",
									},
								},
							}, nil
					},
				},
				appsv1Client:     getAppsV1Client(successfulTestAppsV1Objects),
				oauthv1Client:    fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
				recorder:         setupRecorder(),
				threeScaleClient: getThreeScaleClient(),
				fakeConfig:       getBasicConfigMoc(),
			},
			args: args{
				installation:  getValidInstallation(integreatlyv1alpha1.InstallationTypeMultitenantManagedApi),
				productStatus: &integreatlyv1alpha1.RHMIProductStatus{},
				productConfig: &quota.ProductConfigMock{
					ConfigureFunc: func(obj metav1.Object) error {
						return nil
					},
				},
				uninstall: false,
			},
			assert:      true,
			want:        integreatlyv1alpha1.PhaseCompleted,
			wantErr:     false,
			mockNetwork: true,
		},
		{
			name: "Test successful installation of RHOAM",
			fields: fields{
				sigsClient: moqclient.NewSigsClientMoqWithScheme(scheme, objects...),
				mpm: &marketplace.MarketplaceInterfaceMock{
					InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval, catalogSourceReconciler marketplace.CatalogSourceReconciler) error {

						return nil
					},
					GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (plans *operatorsv1alpha1.InstallPlan, subscription *operatorsv1alpha1.Subscription, e error) {
						return &operatorsv1alpha1.InstallPlan{
								ObjectMeta: metav1.ObjectMeta{
									Name: "3scale-install-plan",
								},
								Status: operatorsv1alpha1.InstallPlanStatus{
									Phase: operatorsv1alpha1.InstallPlanPhaseComplete,
								},
							}, &operatorsv1alpha1.Subscription{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "rhmi-3scale",
									Namespace: "3scale",
								},
								Status: operatorsv1alpha1.SubscriptionStatus{
									Install: &operatorsv1alpha1.InstallPlanReference{
										Name: "3scale-install-plan",
									},
								},
							}, nil
					},
				},
				appsv1Client:     getAppsV1Client(successfulTestAppsV1Objects),
				oauthv1Client:    fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
				recorder:         setupRecorder(),
				threeScaleClient: getThreeScaleClient(),
				fakeConfig:       getBasicConfigMoc(),
			},
			args: args{
				installation:  getValidInstallation(integreatlyv1alpha1.InstallationTypeManagedApi),
				productStatus: &integreatlyv1alpha1.RHMIProductStatus{},
				productConfig: &quota.ProductConfigMock{
					ConfigureFunc: func(obj metav1.Object) error {
						return nil
					},
				},
				uninstall: false,
			},
			assert:      true,
			want:        integreatlyv1alpha1.PhaseCompleted,
			wantErr:     false,
			mockNetwork: true,
		},
		{
			name: "Test successful installation without errors",
			fields: fields{
				sigsClient: moqclient.NewSigsClientMoqWithScheme(scheme, objects...),
				mpm: &marketplace.MarketplaceInterfaceMock{
					InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval, catalogSourceReconciler marketplace.CatalogSourceReconciler) error {

						return nil
					},
					GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (plans *operatorsv1alpha1.InstallPlan, subscription *operatorsv1alpha1.Subscription, e error) {
						return &operatorsv1alpha1.InstallPlan{
								ObjectMeta: metav1.ObjectMeta{
									Name: "3scale-install-plan",
								},
								Status: operatorsv1alpha1.InstallPlanStatus{
									Phase: operatorsv1alpha1.InstallPlanPhaseComplete,
								},
							}, &operatorsv1alpha1.Subscription{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "rhmi-3scale",
									Namespace: "3scale",
								},
								Status: operatorsv1alpha1.SubscriptionStatus{
									Install: &operatorsv1alpha1.InstallPlanReference{
										Name: "3scale-install-plan",
									},
								},
							}, nil
					},
				},
				appsv1Client:     getAppsV1Client(successfulTestAppsV1Objects),
				oauthv1Client:    fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
				recorder:         setupRecorder(),
				threeScaleClient: getThreeScaleClient(),
				fakeConfig:       getBasicConfigMoc(),
			},
			args: args{
				installation:  getValidInstallation(integreatlyv1alpha1.InstallationTypeManagedApi),
				productStatus: &integreatlyv1alpha1.RHMIProductStatus{},
				productConfig: &quota.ProductConfigMock{
					ConfigureFunc: func(obj metav1.Object) error {
						return nil
					},
				},
				uninstall: false,
			},
			assert:      true,
			want:        integreatlyv1alpha1.PhaseCompleted,
			wantErr:     false,
			mockNetwork: true,
		},
		{
			name: "failed to retrieve ingress router ips",
			fields: fields{
				sigsClient: moqclient.NewSigsClientMoqWithScheme(scheme, objects...),
				mpm: &marketplace.MarketplaceInterfaceMock{
					InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval, catalogSourceReconciler marketplace.CatalogSourceReconciler) error {

						return nil
					},
					GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (plans *operatorsv1alpha1.InstallPlan, subscription *operatorsv1alpha1.Subscription, e error) {
						return &operatorsv1alpha1.InstallPlan{
								ObjectMeta: metav1.ObjectMeta{
									Name: "3scale-install-plan",
								},
								Status: operatorsv1alpha1.InstallPlanStatus{
									Phase: operatorsv1alpha1.InstallPlanPhaseComplete,
								},
							}, &operatorsv1alpha1.Subscription{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "rhmi-3scale",
									Namespace: "3scale",
								},
								Status: operatorsv1alpha1.SubscriptionStatus{
									Install: &operatorsv1alpha1.InstallPlanReference{
										Name: "3scale-install-plan",
									},
								},
							}, nil
					},
				},
				appsv1Client:     getAppsV1Client(successfulTestAppsV1Objects),
				oauthv1Client:    fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
				recorder:         setupRecorder(),
				threeScaleClient: getThreeScaleClient(),
				fakeConfig:       getBasicConfigMoc(),
			},
			args: args{
				installation:  getValidInstallation(integreatlyv1alpha1.InstallationTypeManagedApi),
				productStatus: &integreatlyv1alpha1.RHMIProductStatus{},
				productConfig: &quota.ProductConfigMock{
					ConfigureFunc: func(obj metav1.Object) error {
						return nil
					},
				},
				uninstall: false,
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name: "failed to retrieve ingress router service",
			fields: fields{
				sigsClient: func() k8sclient.Client {
					preReqs := getSuccessfullTestPreReqs(integreatlyOperatorNamespace, defaultInstallationNamespace)
					for i, obj := range preReqs {
						routerService, ok := obj.(*corev1.Service)
						if ok && routerService.Name == ingressRouterService.Name && routerService.Namespace == ingressRouterService.Namespace {
							// remove the ingress router service from successful prerequisites
							preReqs = append(preReqs[:i], preReqs[i+1:]...)
						}
					}
					return getSigClient(preReqs, scheme)
				}(),
				mpm: &marketplace.MarketplaceInterfaceMock{
					InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval, catalogSourceReconciler marketplace.CatalogSourceReconciler) error {

						return nil
					},
					GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (plans *operatorsv1alpha1.InstallPlan, subscription *operatorsv1alpha1.Subscription, e error) {
						return &operatorsv1alpha1.InstallPlan{
								ObjectMeta: metav1.ObjectMeta{
									Name: "3scale-install-plan",
								},
								Status: operatorsv1alpha1.InstallPlanStatus{
									Phase: operatorsv1alpha1.InstallPlanPhaseComplete,
								},
							}, &operatorsv1alpha1.Subscription{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "rhmi-3scale",
									Namespace: "3scale",
								},
								Status: operatorsv1alpha1.SubscriptionStatus{
									Install: &operatorsv1alpha1.InstallPlanReference{
										Name: "3scale-install-plan",
									},
								},
							}, nil
					},
				},
				appsv1Client:     getAppsV1Client(successfulTestAppsV1Objects),
				oauthv1Client:    fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
				recorder:         setupRecorder(),
				threeScaleClient: getThreeScaleClient(),
				fakeConfig:       getBasicConfigMoc(),
			},
			args: args{
				installation:  getValidInstallation(integreatlyv1alpha1.InstallationTypeManagedApi),
				productStatus: &integreatlyv1alpha1.RHMIProductStatus{},
				productConfig: &quota.ProductConfigMock{
					ConfigureFunc: func(obj metav1.Object) error {
						return nil
					},
				},
				uninstall: false,
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockNetwork {
				dnsSrv, err := mockDNS("xxx.eu-west-1.elb.amazonaws.com", "127.0.0.1")
				if err != nil {
					t.Fatalf("error mocking dns server: %v", err)
				}
				defer dnsSrv.Close()
				defer mockdns.UnpatchNet(net.DefaultResolver)
				httpSrv, err := mockHTTP("127.0.0.1")
				if err != nil {
					t.Fatalf("error mocking http server: %v", err)
				}
				defer httpSrv.Close()
			}

			r, err := NewReconciler(
				tt.fields.fakeConfig,
				tt.args.installation,
				tt.fields.appsv1Client,
				tt.fields.oauthv1Client,
				tt.fields.threeScaleClient,
				tt.fields.mpm,
				tt.fields.recorder,
				getLogger(),
				localProductDeclaration,
			)
			if err != nil {
				t.Fatalf("Error creating new reconciler %s: %v", constants.ThreeScaleSubscriptionName, err)
			}
			got, err := r.Reconcile(context.TODO(), tt.args.installation, tt.args.productStatus, tt.fields.sigsClient, tt.args.productConfig, tt.args.uninstall)
			if (err != nil) != tt.wantErr {
				t.Errorf("Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Reconcile() got = %v, want %v", got, tt.want)
			}
			if tt.assert {
				err = tt.assertInstallationSuccessful()
				if err != nil {
					t.Fatal(err.Error())
				}
			}
		})
	}
}

func TestReconciler_reconcileBlobStorage(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		ConfigManager config.ConfigReadWriter
		Config        *config.ThreeScale
		mpm           marketplace.MarketplaceInterface
		installation  *integreatlyv1alpha1.RHMI
		tsClient      ThreeScaleInterface
		appsv1Client  appsv1Client.AppsV1Interface
		oauthv1Client oauthClient.OauthV1Interface
		Reconciler    *resources.Reconciler
	}
	type args struct {
		ctx          context.Context
		serverClient k8sclient.Client
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    integreatlyv1alpha1.StatusPhase
		wantErr bool
	}{
		{
			name: "test successful reconcile",
			fields: fields{
				ConfigManager: nil,
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": "test",
				}),
				mpm:           nil,
				installation:  getTestInstallation("managed"),
				tsClient:      nil,
				appsv1Client:  nil,
				oauthv1Client: nil,
				Reconciler:    nil,
			},
			args: args{
				ctx:          context.TODO(),
				serverClient: fake.NewFakeClientWithScheme(scheme, getTestBlobStorage()),
			},
			want:    integreatlyv1alpha1.PhaseCompleted,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				ConfigManager: tt.fields.ConfigManager,
				Config:        tt.fields.Config,
				log:           getLogger(),
				mpm:           tt.fields.mpm,
				installation:  tt.fields.installation,
				tsClient:      tt.fields.tsClient,
				appsv1Client:  tt.fields.appsv1Client,
				oauthv1Client: tt.fields.oauthv1Client,
				Reconciler:    tt.fields.Reconciler,
			}
			got, err := r.reconcileBlobStorage(tt.args.ctx, tt.args.serverClient)
			if (err != nil) != tt.wantErr {
				t.Errorf("reconcileBlobStorage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("reconcileBlobStorage() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_reconcileComponents(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		ConfigManager config.ConfigReadWriter
		Config        *config.ThreeScale
		mpm           marketplace.MarketplaceInterface
		installation  *integreatlyv1alpha1.RHMI
		tsClient      ThreeScaleInterface
		appsv1Client  appsv1Client.AppsV1Interface
		oauthv1Client oauthClient.OauthV1Interface
		Reconciler    *resources.Reconciler
		productConfig *quota.ProductConfigMock
	}
	type args struct {
		ctx          context.Context
		serverClient k8sclient.Client
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    integreatlyv1alpha1.StatusPhase
		wantErr bool
	}{
		{
			name: "test successful reconcile of s3 blob storage, non STS mode",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{
					GetOperatorNamespaceFunc: func() string {
						return "redhat-rhoam-operator"
					},
				},
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": "test",
				}),
				installation:  getTestInstallation("managed"),
				productConfig: productConfigMock(),
			},
			args: args{
				ctx: context.TODO(),
				serverClient: fake.NewFakeClientWithScheme(scheme, getTestBlobStorage(),
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test",
							Namespace: "test",
						},
						Data: map[string][]byte{
							"credentialKeyID":     []byte("test"),
							"credentialSecretKey": []byte("test"),
							"bucketName":          []byte("test"),
							"bucketRegion":        []byte("test"),
							"minio":               []byte("test"),
						},
					},
					&threescalev1.APIManager{
						TypeMeta: metav1.TypeMeta{},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "3scale",
							Namespace: "test",
						},
						Spec:   threescalev1.APIManagerSpec{},
						Status: threescalev1.APIManagerStatus{},
					},
					&cloudcredentialv1.CloudCredential{
						ObjectMeta: metav1.ObjectMeta{
							Name: sts.ClusterCloudCredentialName,
						},
						Spec: cloudcredentialv1.CloudCredentialSpec{
							CredentialsMode: cloudcredentialv1.CloudCredentialsModeDefault,
						},
					},
					&configv1.Infrastructure{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cluster",
						},
						Status: configv1.InfrastructureStatus{
							PlatformStatus: &configv1.PlatformStatus{
								Type: configv1.AWSPlatformType,
							},
						},
					}),
			},
			want:    integreatlyv1alpha1.PhaseInProgress,
			wantErr: false,
		},
		{
			name: "test successful reconcile of s3 blob storage, STS mode",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{
					GetOperatorNamespaceFunc: func() string {
						return "redhat-rhoam-operator"
					},
				},
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": "test",
				}),
				installation:  getTestInstallation("managed"),
				productConfig: productConfigMock(),
			},
			args: args{
				ctx: context.TODO(),
				serverClient: fake.NewFakeClientWithScheme(scheme, getTestBlobStorage(),
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test",
							Namespace: "test",
						},
						Data: map[string][]byte{
							"bucketName":   []byte("test"),
							"bucketRegion": []byte("test"),
						},
					},
					&threescalev1.APIManager{
						TypeMeta: metav1.TypeMeta{},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "3scale",
							Namespace: "test",
						},
						Spec:   threescalev1.APIManagerSpec{},
						Status: threescalev1.APIManagerStatus{},
					},
					&cloudcredentialv1.CloudCredential{
						ObjectMeta: metav1.ObjectMeta{
							Name: sts.ClusterCloudCredentialName,
						},
						Spec: cloudcredentialv1.CloudCredentialSpec{
							CredentialsMode: cloudcredentialv1.CloudCredentialsModeManual,
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      stsS3CredentialsSecretName,
							Namespace: "test",
						},
						Data: map[string][]byte{
							"role_arn": []byte("role"),
						},
					}),
			},
			want:    integreatlyv1alpha1.PhaseInProgress,
			wantErr: false,
		},
		{
			name: "test Unsuccessful reconcile of s3 blob storage, STS mode, Error getting sts secret",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{
					GetOperatorNamespaceFunc: func() string {
						return "null"
					},
				},
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": "test",
				}),
				installation: getTestInstallation("managed"),
			},
			args: args{
				ctx: context.TODO(),
				serverClient: fake.NewFakeClientWithScheme(scheme, getTestBlobStorage(),
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test",
							Namespace: "test",
						},
						Data: map[string][]byte{
							"credentialKeyID": []byte("test"),
						},
					},
					&threescalev1.APIManager{
						TypeMeta: metav1.TypeMeta{},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "3scale",
							Namespace: "test",
						},
						Spec:   threescalev1.APIManagerSpec{},
						Status: threescalev1.APIManagerStatus{},
					},
					&cloudcredentialv1.CloudCredential{
						ObjectMeta: metav1.ObjectMeta{
							Name: sts.ClusterCloudCredentialName,
						},
						Spec: cloudcredentialv1.CloudCredentialSpec{
							CredentialsMode: cloudcredentialv1.CloudCredentialsModeManual,
						},
					}),
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name: "Error in STS mode checking",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{
					GetOperatorNamespaceFunc: func() string {
						return "redhat-rhoam-operator"
					},
				},
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": "test",
				}),
				installation: getTestInstallation("managed"),
			},
			args: args{
				ctx: context.TODO(),
				serverClient: fake.NewFakeClientWithScheme(scheme, getTestBlobStorage(),
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test",
							Namespace: "test",
						},
					},
					&cloudcredentialv1.CloudCredential{
						ObjectMeta: metav1.ObjectMeta{
							Name: "not-exists",
						},
						Spec: cloudcredentialv1.CloudCredentialSpec{
							CredentialsMode: cloudcredentialv1.CloudCredentialsModeManual,
						},
					}),
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name: "test successful reconcile of s3 blob storage, GCP",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{
					GetOperatorNamespaceFunc: func() string {
						return "redhat-rhoam-operator"
					},
				},
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": "test",
				}),
				installation:  getTestInstallation("managed"),
				productConfig: productConfigMock(),
			},
			args: args{
				ctx: context.TODO(),
				serverClient: fake.NewFakeClientWithScheme(scheme, getTestBlobStorage(),
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test",
							Namespace: "test",
						},
						Data: map[string][]byte{
							"credentialKeyID":     []byte("test"),
							"credentialSecretKey": []byte("test"),
							"bucketName":          []byte("test"),
							"bucketRegion":        []byte("test"),
							"minio":               []byte("test"),
						},
					},
					&threescalev1.APIManager{
						TypeMeta: metav1.TypeMeta{},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "3scale",
							Namespace: "test",
						},
						Spec:   threescalev1.APIManagerSpec{},
						Status: threescalev1.APIManagerStatus{},
					},
					&cloudcredentialv1.CloudCredential{
						ObjectMeta: metav1.ObjectMeta{
							Name: sts.ClusterCloudCredentialName,
						},
						Spec: cloudcredentialv1.CloudCredentialSpec{
							CredentialsMode: cloudcredentialv1.CloudCredentialsModeDefault,
						},
					},
					&configv1.Infrastructure{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cluster",
						},
						Status: configv1.InfrastructureStatus{
							PlatformStatus: &configv1.PlatformStatus{
								Type: configv1.GCPPlatformType,
							},
						},
					},
					&noobaav1.ObjectBucketClaim{
						ObjectMeta: metav1.ObjectMeta{
							Name:      mcg.ThreescaleBucketClaim,
							Namespace: mcg.DefaultInstallationNamespace + "-operator",
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      mcg.ThreescaleBucketClaim,
							Namespace: mcg.DefaultInstallationNamespace + "-operator",
						},
						Data: map[string][]byte{},
					},
					&routev1.Route{
						ObjectMeta: metav1.ObjectMeta{
							Name:      s3RouteName,
							Namespace: mcg.DefaultInstallationNamespace + "-operator",
						},
					}),
			},
			want:    integreatlyv1alpha1.PhaseInProgress,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				ConfigManager: tt.fields.ConfigManager,
				Config:        tt.fields.Config,
				log:           getLogger(),
				mpm:           tt.fields.mpm,
				installation:  tt.fields.installation,
				tsClient:      tt.fields.tsClient,
				appsv1Client:  tt.fields.appsv1Client,
				oauthv1Client: tt.fields.oauthv1Client,
				Reconciler:    tt.fields.Reconciler,
			}
			got, err := r.reconcileComponents(tt.args.ctx, tt.args.serverClient, tt.fields.productConfig)
			if (err != nil) != tt.wantErr {
				t.Errorf("reconcileComponents() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("reconcileComponents() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_syncOpenshiftAdmimMembership(t *testing.T) {
	calledSetUserAsAdmin := false

	tsClientMock := ThreeScaleInterfaceMock{
		SetUserAsAdminFunc: func(userID int, accessToken string) (*http.Response, error) {
			if userID != 1 {
				t.Fatalf("Unexpected user promoted to admin. Expected User with ID 1, got user with ID %d", userID)
			} else {
				calledSetUserAsAdmin = true
			}

			return &http.Response{
				StatusCode: 200,
			}, nil
		},
		SetUserAsMemberFunc: func(userID int, accessToken string) (*http.Response, error) {
			t.Fatalf("Unexpected call to `SetUserAsMember`. Called with userID %d", userID)

			return &http.Response{
				StatusCode: 200,
			}, nil
		},
	}

	openshiftAdminGroup := &usersv1.Group{
		Users: usersv1.OptionalNames{
			"user1",
			"user2",
		},
	}

	newTsUsers := &Users{
		Users: []*User{
			{
				UserDetails: UserDetails{
					Id:   1,
					Role: memberRole,
					// User is in OS admin group. Should be promoted
					Username: "User1",
				},
			},
			{
				UserDetails: UserDetails{
					Id:   2,
					Role: adminRole,
					// User is in OS admin group and admin in 3scale. Should
					// be ignored
					Username: "User2",
				},
			},
			{
				UserDetails{
					Id:   3,
					Role: adminRole,
					// User is not in OS admin group but is already admin.
					// Should NOT be demoted
					Username: "User3",
				},
			},
		},
	}

	err := syncOpenshiftAdminMembership(openshiftAdminGroup, newTsUsers, "", &tsClientMock, "")

	if err != nil {
		t.Fatalf("Unexpected error when reconcilling openshift admin membership: %s", err)
	}

	if !calledSetUserAsAdmin {
		t.Fatal("Expected user with ID 1 to be promoted as admin, but no promotion was invoked")
	}
}

func TestReconciler_ensureDeploymentConfigsReady(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		ConfigManager config.ConfigReadWriter
		Config        *config.ThreeScale
		mpm           marketplace.MarketplaceInterface
		installation  *integreatlyv1alpha1.RHMI
		tsClient      ThreeScaleInterface
		appsv1Client  appsv1Client.AppsV1Interface
		oauthv1Client oauthClient.OauthV1Interface
		Reconciler    *resources.Reconciler
		extraParams   map[string]string
		recorder      record.EventRecorder
		log           l.Logger
	}
	type args struct {
		ctx              context.Context
		serverClient     k8sclient.Client
		productNamespace string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    integreatlyv1alpha1.StatusPhase
		wantErr bool
	}{
		{
			name: "Test - Unable to get deployment config - PhaseFailed",
			args: args{
				ctx: context.TODO(),
				serverClient: &moqclient.SigsClientInterfaceMock{GetFunc: func(ctx context.Context, key k8sTypes.NamespacedName, obj k8sclient.Object) error {
					return fmt.Errorf("fetch error")
				}},
				productNamespace: defaultInstallationNamespace,
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name: "Test - Deployment config has a 'False' condition - Rollout success - PhaseCreatingComponents",
			fields: fields{
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": defaultInstallationNamespace,
				}),
				appsv1Client: getAppsV1Client(successfulTestAppsV1Objects),
				log:          getLogger(),
			},
			args: args{
				ctx: context.TODO(),
				serverClient: fake.NewFakeClientWithScheme(scheme,
					&openshiftappsv1.DeploymentConfig{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "apicast-production",
							Namespace: defaultInstallationNamespace,
							Labels: map[string]string{
								"app": "apicast-production",
							},
						},
						Status: openshiftappsv1.DeploymentConfigStatus{
							Conditions: []openshiftappsv1.DeploymentCondition{
								{
									Status: corev1.ConditionFalse,
								},
							},
						},
					},
				),
				productNamespace: defaultInstallationNamespace,
			},
			want: integreatlyv1alpha1.PhaseCreatingComponents,
		},
		{
			name: "Test - Waiting for replicas to be rolled out - Condition Unknown - PhaseInProgress",
			fields: fields{
				log: getLogger(),
			},
			args: args{
				ctx: context.TODO(),
				serverClient: fake.NewFakeClientWithScheme(scheme,
					&openshiftappsv1.DeploymentConfig{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "apicast-production",
							Namespace: defaultInstallationNamespace,
							Labels: map[string]string{
								"app": "apicast-production",
							},
						},
						Status: openshiftappsv1.DeploymentConfigStatus{
							Conditions: []openshiftappsv1.DeploymentCondition{
								{
									Status: corev1.ConditionUnknown,
								},
							},
						},
					},
				),
				productNamespace: defaultInstallationNamespace,
			},
			want:    integreatlyv1alpha1.PhaseInProgress,
			wantErr: true,
		},
		{
			name: "Test - Waiting for replicas to be rolled out - Replicas != AvailableReplicas - PhaseInProgress",
			fields: fields{
				log: getLogger(),
			},
			args: args{
				ctx: context.TODO(),
				serverClient: fake.NewFakeClientWithScheme(scheme,
					&openshiftappsv1.DeploymentConfig{
						TypeMeta: metav1.TypeMeta{},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "apicast-production",
							Namespace: defaultInstallationNamespace,
							Labels: map[string]string{
								"app": "apicast-production",
							},
						},
						Status: openshiftappsv1.DeploymentConfigStatus{
							Conditions: []openshiftappsv1.DeploymentCondition{
								{
									Status: corev1.ConditionTrue,
								},
							},
							Replicas:      1,
							ReadyReplicas: 0,
						},
					},
				),
				productNamespace: defaultInstallationNamespace,
			},
			want:    integreatlyv1alpha1.PhaseInProgress,
			wantErr: true,
		},
		{
			name: "Test - Waiting for replicas to be rolled out - ReadyReplicas != UpdatedReplicas - PhaseInProgress",
			fields: fields{
				log: getLogger(),
			},
			args: args{
				ctx: context.TODO(),
				serverClient: fake.NewFakeClientWithScheme(scheme,
					&openshiftappsv1.DeploymentConfig{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "apicast-production",
							Namespace: defaultInstallationNamespace,
							Labels: map[string]string{
								"app": "apicast-production",
							},
						},
						Status: openshiftappsv1.DeploymentConfigStatus{
							Conditions: []openshiftappsv1.DeploymentCondition{
								{
									Status: corev1.ConditionTrue,
								},
							},
							ReadyReplicas:   1,
							UpdatedReplicas: 0,
						},
					},
				),
				productNamespace: defaultInstallationNamespace,
			},
			want:    integreatlyv1alpha1.PhaseInProgress,
			wantErr: true,
		},
		{
			name: "Test - Replicas all rolled out - PhaseComplete",
			args: args{
				ctx:              context.TODO(),
				serverClient:     fake.NewFakeClientWithScheme(scheme, getSuccessfullTestPreReqs(integreatlyOperatorNamespace, defaultInstallationNamespace)...),
				productNamespace: defaultInstallationNamespace,
			},
			want: integreatlyv1alpha1.PhaseCompleted,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				ConfigManager: tt.fields.ConfigManager,
				Config:        tt.fields.Config,
				mpm:           tt.fields.mpm,
				installation:  tt.fields.installation,
				tsClient:      tt.fields.tsClient,
				appsv1Client:  tt.fields.appsv1Client,
				oauthv1Client: tt.fields.oauthv1Client,
				Reconciler:    tt.fields.Reconciler,
				extraParams:   tt.fields.extraParams,
				recorder:      tt.fields.recorder,
				log:           tt.fields.log,
			}
			got, err := r.ensureDeploymentConfigsReady(tt.args.ctx, tt.args.serverClient, tt.args.productNamespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("ensureDeploymentConfigsReady() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ensureDeploymentConfigsReady() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_reconcileOpenshiftUsers(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		ConfigManager config.ConfigReadWriter
		Config        *config.ThreeScale
		mpm           marketplace.MarketplaceInterface
		installation  *integreatlyv1alpha1.RHMI
		tsClient      ThreeScaleInterface
		appsv1Client  appsv1Client.AppsV1Interface
		oauthv1Client oauthClient.OauthV1Interface
		Reconciler    *resources.Reconciler
		extraParams   map[string]string
		recorder      record.EventRecorder
		log           l.Logger
	}
	type args struct {
		ctx          context.Context
		installation *integreatlyv1alpha1.RHMI
		serverClient k8sclient.Client
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    integreatlyv1alpha1.StatusPhase
		wantErr bool
	}{
		{
			name: "Test - Read RHSSO Config failed - PhaseFailed",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{ReadRHSSOFunc: func() (*config.RHSSO, error) {
					return nil, fmt.Errorf("read error")
				}},
				log: getLogger(),
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name: "Test - System seed secret failed - PhaseFailed",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{ReadRHSSOFunc: func() (*config.RHSSO, error) {
					return config.NewRHSSO(config.ProductConfig{
						"NAMESPACE": "rhsso",
					}), nil
				}},
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": defaultInstallationNamespace,
				}),
				log: getLogger(),
			},
			args: args{
				serverClient: &moqclient.SigsClientInterfaceMock{
					GetFunc: func(ctx context.Context, key k8sTypes.NamespacedName, obj k8sclient.Object) error {
						return fmt.Errorf("get error")
					},
				},
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name: "Test - Get Keycloak users failed - PhaseFailed",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{ReadRHSSOFunc: func() (*config.RHSSO, error) {
					return config.NewRHSSO(config.ProductConfig{
						"NAMESPACE": "rhsso",
					}), nil
				}},
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": defaultInstallationNamespace,
				}),
				log: getLogger(),
			},
			args: args{
				serverClient: &moqclient.SigsClientInterfaceMock{
					GetFunc: func(ctx context.Context, key k8sTypes.NamespacedName, obj k8sclient.Object) error {
						return nil
					},
					ListFunc: func(ctx context.Context, list k8sclient.ObjectList, opts ...k8sclient.ListOption) error {
						return fmt.Errorf("list error")
					},
				},
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name: "Test - Get 3scale Users failed - PhaseInProgress",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{ReadRHSSOFunc: func() (*config.RHSSO, error) {
					return config.NewRHSSO(config.ProductConfig{
						"NAMESPACE": "rhsso",
					}), nil
				}},
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": defaultInstallationNamespace,
				}),
				log: getLogger(),
				tsClient: &ThreeScaleInterfaceMock{
					GetUsersFunc: func(accessToken string) (*Users, error) {
						return nil, fmt.Errorf("get error")
					},
				},
			},
			args: args{
				serverClient: &moqclient.SigsClientInterfaceMock{
					GetFunc: func(ctx context.Context, key k8sTypes.NamespacedName, obj k8sclient.Object) error {
						return nil
					},
					ListFunc: func(ctx context.Context, list k8sclient.ObjectList, opts ...k8sclient.ListOption) error {
						return nil
					},
				},
			},
			want:    integreatlyv1alpha1.PhaseInProgress,
			wantErr: true,
		},
		{
			name: "Test - Reconcile Successful - PhaseComplete",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{ReadRHSSOFunc: func() (*config.RHSSO, error) {
					return config.NewRHSSO(config.ProductConfig{
						"NAMESPACE": "rhsso",
					}), nil
				}},
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": defaultInstallationNamespace,
				}),
				log: getLogger(),
				tsClient: &ThreeScaleInterfaceMock{
					GetUsersFunc: func(accessToken string) (*Users, error) {
						return &Users{
							Users: []*User{
								{
									UserDetails: UserDetails{
										Username: "updated-3scale",
										Id:       1,
									},
								},
								{
									UserDetails: UserDetails{
										Username: "notInKeyCloak",
									},
								},
							},
						}, nil
					},
					AddUserFunc: func(username string, email string, password string, accessToken string) (*http.Response, error) {
						return &http.Response{
							StatusCode: http.StatusOK,
						}, nil
					},
					DeleteUserFunc: func(userID int, accessToken string) (*http.Response, error) {
						return &http.Response{
							StatusCode: http.StatusOK,
						}, nil
					},
					UpdateUserFunc: func(userID int, username string, email string, accessToken string) (*http.Response, error) {
						return &http.Response{
							StatusCode: http.StatusOK,
						}, nil
					},
					GetUserFunc: func(username string, accessToken string) (*User, error) {
						return &User{
							UserDetails: UserDetails{
								Username: defaultInstallationNamespace,
								Id:       1,
							},
						}, nil
					},
				},
			},
			args: args{
				serverClient: fake.NewFakeClientWithScheme(scheme,
					append(getSuccessfullTestPreReqs(integreatlyOperatorNamespace, defaultInstallationNamespace),
						&keycloak.KeycloakUser{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "generated-3scale",
								Namespace: "rhsso",
								Labels: map[string]string{
									"sso": "integreatly",
								},
							},
							Spec: keycloak.KeycloakUserSpec{
								User: keycloak.KeycloakAPIUser{
									UserName: defaultInstallationNamespace,
									Attributes: map[string][]string{
										user3ScaleID: {fmt.Sprint(1)},
									},
								},
							},
						},
					)...),
				installation: getValidInstallation(integreatlyv1alpha1.InstallationTypeManagedApi),
			},
			want: integreatlyv1alpha1.PhaseCompleted,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				ConfigManager: tt.fields.ConfigManager,
				Config:        tt.fields.Config,
				mpm:           tt.fields.mpm,
				installation:  tt.fields.installation,
				tsClient:      tt.fields.tsClient,
				appsv1Client:  tt.fields.appsv1Client,
				oauthv1Client: tt.fields.oauthv1Client,
				Reconciler:    tt.fields.Reconciler,
				extraParams:   tt.fields.extraParams,
				recorder:      tt.fields.recorder,
				log:           tt.fields.log,
			}
			got, err := r.reconcileOpenshiftUsers(tt.args.ctx, tt.args.installation, tt.args.serverClient)
			if (err != nil) != tt.wantErr {
				t.Errorf("reconcileOpenshiftUsers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("reconcileOpenshiftUsers() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_updateKeycloakUsersAttributeWith3ScaleUserId(t *testing.T) {
	accessToken := "accessToken"

	type fields struct {
		ConfigManager config.ConfigReadWriter
		Config        *config.ThreeScale
		mpm           marketplace.MarketplaceInterface
		installation  *integreatlyv1alpha1.RHMI
		tsClient      ThreeScaleInterface
		appsv1Client  appsv1Client.AppsV1Interface
		oauthv1Client oauthClient.OauthV1Interface
		Reconciler    *resources.Reconciler
		extraParams   map[string]string
		recorder      record.EventRecorder
		log           l.Logger
	}
	type args struct {
		ctx          context.Context
		serverClient k8sclient.Client
		kcu          []keycloak.KeycloakAPIUser
		accessToken  *string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    integreatlyv1alpha1.StatusPhase
		wantErr bool
	}{
		{
			name: "Test - Read RHSSO Config failed - PhaseFailed",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{ReadRHSSOFunc: func() (*config.RHSSO, error) {
					return nil, fmt.Errorf("read error")
				}},
				log: getLogger(),
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name: "Test - Read RHSSO Config failed - PhaseFailed",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{ReadRHSSOFunc: func() (*config.RHSSO, error) {
					return nil, fmt.Errorf("read error")
				}},
				log: getLogger(),
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name: "Test - Get 3scale User failed - Continued - PhaseCompleted",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{ReadRHSSOFunc: func() (*config.RHSSO, error) {
					return config.NewRHSSO(config.ProductConfig{
						"NAMESPACE": "rhsso",
					}), nil
				}},
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": defaultInstallationNamespace,
				}),
				log: getLogger(),
				tsClient: &ThreeScaleInterfaceMock{
					GetUserFunc: func(username string, accessToken string) (*User, error) {
						return nil, fmt.Errorf("get error")
					},
				},
			},
			args: args{
				kcu: []keycloak.KeycloakAPIUser{
					{
						UserName: defaultInstallationNamespace,
					},
				},
				accessToken: &accessToken,
			},
			want: integreatlyv1alpha1.PhaseCompleted,
		},
		{
			name: "Test - Update Keycloak User failed - PhaseInProgress",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{ReadRHSSOFunc: func() (*config.RHSSO, error) {
					return config.NewRHSSO(config.ProductConfig{
						"NAMESPACE": "rhsso",
					}), nil
				}},
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": defaultInstallationNamespace,
				}),
				log: getLogger(),
				tsClient: &ThreeScaleInterfaceMock{
					GetUserFunc: func(username string, accessToken string) (*User, error) {
						return &User{UserDetails: UserDetails{Id: 1}}, nil
					},
				},
			},
			args: args{
				kcu: []keycloak.KeycloakAPIUser{
					{
						UserName: defaultInstallationNamespace,
					},
				},
				accessToken: &accessToken,
				serverClient: &moqclient.SigsClientInterfaceMock{
					GetFunc: func(ctx context.Context, key k8sTypes.NamespacedName, obj k8sclient.Object) error {
						return fmt.Errorf("get error")
					},
				},
			},
			want:    integreatlyv1alpha1.PhaseInProgress,
			wantErr: true,
		},
		{
			name: "Test - Update Keycloak User successful - PhaseComplete",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{ReadRHSSOFunc: func() (*config.RHSSO, error) {
					return config.NewRHSSO(config.ProductConfig{
						"NAMESPACE": "rhsso",
					}), nil
				}},
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": defaultInstallationNamespace,
				}),
				log: getLogger(),
				tsClient: &ThreeScaleInterfaceMock{
					GetUserFunc: func(username string, accessToken string) (*User, error) {
						return &User{UserDetails: UserDetails{Id: 1}}, nil
					},
				},
			},
			args: args{
				kcu: []keycloak.KeycloakAPIUser{
					{
						UserName: "test",
					},
				},
				accessToken: &accessToken,
				serverClient: &moqclient.SigsClientInterfaceMock{
					GetFunc: func(ctx context.Context, key k8sTypes.NamespacedName, obj k8sclient.Object) error {
						return nil
					},
					UpdateFunc: func(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.UpdateOption) error {
						return nil
					},
				},
			},
			want: integreatlyv1alpha1.PhaseCompleted,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				ConfigManager: tt.fields.ConfigManager,
				Config:        tt.fields.Config,
				mpm:           tt.fields.mpm,
				installation:  tt.fields.installation,
				tsClient:      tt.fields.tsClient,
				appsv1Client:  tt.fields.appsv1Client,
				oauthv1Client: tt.fields.oauthv1Client,
				Reconciler:    tt.fields.Reconciler,
				extraParams:   tt.fields.extraParams,
				recorder:      tt.fields.recorder,
				log:           tt.fields.log,
			}
			got, err := r.updateKeycloakUsersAttributeWith3ScaleUserId(tt.args.ctx, tt.args.serverClient, tt.args.kcu, tt.args.accessToken)
			if (err != nil) != tt.wantErr {
				t.Errorf("updateKeycloakUsersAttributeWith3ScaleUserId() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("updateKeycloakUsersAttributeWith3ScaleUserId() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_getUserDiff(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		ConfigManager config.ConfigReadWriter
		Config        *config.ThreeScale
		mpm           marketplace.MarketplaceInterface
		installation  *integreatlyv1alpha1.RHMI
		tsClient      ThreeScaleInterface
		appsv1Client  appsv1Client.AppsV1Interface
		oauthv1Client oauthClient.OauthV1Interface
		Reconciler    *resources.Reconciler
		extraParams   map[string]string
		recorder      record.EventRecorder
		log           l.Logger
	}
	type args struct {
		ctx          context.Context
		serverClient k8sclient.Client
		kcUsers      []keycloak.KeycloakAPIUser
		tsUsers      []*User
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []keycloak.KeycloakAPIUser
		want1  []*User
		want2  []*User
	}{
		{
			name: "Test - Read RHSSO Config failed - PhaseFailed",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{ReadRHSSOFunc: func() (*config.RHSSO, error) {
					return nil, fmt.Errorf("read error")
				}},
				log: getLogger(),
			},
			want:  nil,
			want1: nil,
			want2: nil,
		},
		{
			name: "Test - Keycloak User not in 3scale appended to added",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{ReadRHSSOFunc: func() (*config.RHSSO, error) {
					return config.NewRHSSO(config.ProductConfig{
						"NAMESPACE": "rhsso",
					}), nil
				}},
				log: getLogger(),
			},
			args: args{
				tsUsers: []*User{
					{
						UserDetails: UserDetails{
							Username: defaultInstallationNamespace,
						},
					},
				},
				kcUsers: []keycloak.KeycloakAPIUser{
					{
						UserName: "NEW-3SCALE",
					},
					{
						UserName: defaultInstallationNamespace,
					},
				},
			},
			want: []keycloak.KeycloakAPIUser{
				{
					UserName: "NEW-3SCALE",
				},
			},
			want1: nil,
			want2: nil,
		},
		{
			name: "Test - 3scale User not in Keycloak appended to deleted & comparison is case sensitive",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{ReadRHSSOFunc: func() (*config.RHSSO, error) {
					return config.NewRHSSO(config.ProductConfig{
						"NAMESPACE": "rhsso",
					}), nil
				}},
				log: getLogger(),
			},
			args: args{
				tsUsers: []*User{
					{
						UserDetails: UserDetails{
							Username: defaultInstallationNamespace,
						},
					},
					{
						UserDetails: UserDetails{
							Username: "3SCALE",
						},
					},
				},
				kcUsers: []keycloak.KeycloakAPIUser{
					{
						UserName: defaultInstallationNamespace,
					},
				},
				serverClient: fake.NewFakeClientWithScheme(scheme, &keycloak.KeycloakUser{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "generated-3scale",
						Namespace: "rhsso",
					},
				}),
			},
			want: nil,
			want1: []*User{
				{
					UserDetails: UserDetails{
						Username: "3SCALE",
					},
				},
			},
			want2: nil,
		},
		{
			name: "Test - Get keycloak user for deletion failed - appended to deleted",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{ReadRHSSOFunc: func() (*config.RHSSO, error) {
					return config.NewRHSSO(config.ProductConfig{
						"NAMESPACE": "rhsso",
					}), nil
				}},
				log: getLogger(),
			},
			args: args{
				tsUsers: []*User{
					{
						UserDetails: UserDetails{
							Username: defaultInstallationNamespace,
						},
					},
					{
						UserDetails: UserDetails{
							Username: "notInKeyCloak",
						},
					},
				},
				kcUsers: []keycloak.KeycloakAPIUser{
					{
						UserName: defaultInstallationNamespace,
					},
				},
				serverClient: fake.NewFakeClientWithScheme(scheme),
			},
			want: nil,
			want1: []*User{
				{
					UserDetails: UserDetails{
						Username: "notInKeyCloak",
					},
				},
			},
			want2: nil,
		},
		{
			name: "Test - 3scale User updated appended to updated and to added",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{ReadRHSSOFunc: func() (*config.RHSSO, error) {
					return config.NewRHSSO(config.ProductConfig{
						"NAMESPACE": "rhsso",
					}), nil
				}},
				log: getLogger(),
			},
			args: args{
				tsUsers: []*User{
					{
						UserDetails: UserDetails{
							Username: fmt.Sprintf("Updated-%s", defaultInstallationNamespace),
							Id:       1,
						},
					},
				},
				kcUsers: []keycloak.KeycloakAPIUser{
					{
						UserName: defaultInstallationNamespace,
					},
				},
				serverClient: fake.NewFakeClientWithScheme(scheme, &keycloak.KeycloakUser{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("generated-%s", defaultInstallationNamespace),
						Namespace: "rhsso",
					},
					Spec: keycloak.KeycloakUserSpec{
						User: keycloak.KeycloakAPIUser{
							Attributes: map[string][]string{
								user3ScaleID: {fmt.Sprint(1)},
							},
						},
					},
				}),
			},
			want: []keycloak.KeycloakAPIUser{
				{
					UserName: defaultInstallationNamespace,
				},
			},
			want1: nil,
			want2: []*User{
				{
					UserDetails: UserDetails{
						Username: fmt.Sprintf("Updated-%s", defaultInstallationNamespace),
						Id:       1,
					},
				},
			},
		},
		{
			name: "Test - 3scale User with same uppercase name in KeyCloak CR appended to updated and to added",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{ReadRHSSOFunc: func() (*config.RHSSO, error) {
					return config.NewRHSSO(config.ProductConfig{
						"NAMESPACE": "rhsso",
					}), nil
				}},
				log: getLogger(),
			},
			args: args{
				tsUsers: []*User{
					{
						UserDetails: UserDetails{
							Username: fmt.Sprintf("UpperCase-%s", defaultInstallationNamespace),
							Id:       1,
						},
					},
				},
				kcUsers: []keycloak.KeycloakAPIUser{
					{
						UserName: fmt.Sprintf("UpperCase-%s", defaultInstallationNamespace),
						Attributes: map[string][]string{
							user3ScaleID: {fmt.Sprint(1)},
						},
					},
				},
				serverClient: fake.NewFakeClientWithScheme(scheme, &keycloak.KeycloakUser{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("generated-uppercase-%s", defaultInstallationNamespace),
						Namespace: "rhsso",
					},
					Spec: keycloak.KeycloakUserSpec{
						User: keycloak.KeycloakAPIUser{
							Attributes: map[string][]string{
								user3ScaleID: {fmt.Sprint(1)},
							},
							UserName: fmt.Sprintf("UpperCase-%s", defaultInstallationNamespace),
						},
					},
				}),
			},
			want: []keycloak.KeycloakAPIUser{
				{
					UserName: fmt.Sprintf("UpperCase-%s", defaultInstallationNamespace),
					Attributes: map[string][]string{
						user3ScaleID: {fmt.Sprint(1)},
					},
				},
			},
			want1: nil,
			want2: []*User{
				{
					UserDetails: UserDetails{
						Username: fmt.Sprintf("UpperCase-%s", defaultInstallationNamespace),
						Id:       1,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				ConfigManager: tt.fields.ConfigManager,
				Config:        tt.fields.Config,
				mpm:           tt.fields.mpm,
				installation:  tt.fields.installation,
				tsClient:      tt.fields.tsClient,
				appsv1Client:  tt.fields.appsv1Client,
				oauthv1Client: tt.fields.oauthv1Client,
				Reconciler:    tt.fields.Reconciler,
				extraParams:   tt.fields.extraParams,
				recorder:      tt.fields.recorder,
				log:           tt.fields.log,
			}
			got, got1, got2 := r.getUserDiff(tt.args.ctx, tt.args.serverClient, tt.args.kcUsers, tt.args.tsUsers)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getUserDiff() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("getUserDiff() got1 = %v, want %v", got1, tt.want1)
			}
			if !reflect.DeepEqual(got2, tt.want2) {
				t.Errorf("getUserDiff() got2 = %v, want %v", got2, tt.want2)
			}
		})
	}
}

func TestReconciler_findCustomDomainCr(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		ConfigManager config.ConfigReadWriter
		Config        *config.ThreeScale
		mpm           marketplace.MarketplaceInterface
		installation  *integreatlyv1alpha1.RHMI
		tsClient      ThreeScaleInterface
		appsv1Client  appsv1Client.AppsV1Interface
		oauthv1Client oauthClient.OauthV1Interface
		Reconciler    *resources.Reconciler
		extraParams   map[string]string
		recorder      record.EventRecorder
		log           l.Logger
	}
	type args struct {
		ctx          context.Context
		serverClient k8sclient.Client
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    integreatlyv1alpha1.StatusPhase
		wantErr bool
	}{
		{
			name: "Found custom domain CR",
			fields: fields{
				installation: getValidInstallation(integreatlyv1alpha1.InstallationTypeManagedApi),
			},
			args: args{
				ctx: context.TODO(),
				serverClient: fake.NewFakeClientWithScheme(scheme, &customdomainv1alpha1.CustomDomainList{
					Items: []customdomainv1alpha1.CustomDomain{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "goodDomain",
							},
							Spec: customdomainv1alpha1.CustomDomainSpec{
								Domain: "apps.example.com",
							},
							Status: customdomainv1alpha1.CustomDomainStatus{
								State: "Ready",
							},
						},
					},
				}),
			},
			want:    integreatlyv1alpha1.PhaseCompleted,
			wantErr: false,
		},
		{
			name: "Error finding custom domain CR",
			fields: fields{
				installation: getValidInstallation(integreatlyv1alpha1.InstallationTypeManagedApi),
			},
			args: args{
				ctx:          context.TODO(),
				serverClient: fake.NewFakeClientWithScheme(scheme),
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				ConfigManager: tt.fields.ConfigManager,
				Config:        tt.fields.Config,
				mpm:           tt.fields.mpm,
				installation:  tt.fields.installation,
				tsClient:      tt.fields.tsClient,
				appsv1Client:  tt.fields.appsv1Client,
				oauthv1Client: tt.fields.oauthv1Client,
				Reconciler:    tt.fields.Reconciler,
				extraParams:   tt.fields.extraParams,
				recorder:      tt.fields.recorder,
				log:           tt.fields.log,
			}
			got, err := r.findCustomDomainCr(tt.args.ctx, tt.args.serverClient)
			if (err != nil) != tt.wantErr {
				t.Errorf("findCustomDomainCr() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("findCustomDomainCr() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_useCustomDomain(t *testing.T) {
	type fields struct {
		ConfigManager config.ConfigReadWriter
		Config        *config.ThreeScale
		mpm           marketplace.MarketplaceInterface
		installation  *integreatlyv1alpha1.RHMI
		tsClient      ThreeScaleInterface
		appsv1Client  appsv1Client.AppsV1Interface
		oauthv1Client oauthClient.OauthV1Interface
		Reconciler    *resources.Reconciler
		extraParams   map[string]string
		recorder      record.EventRecorder
		log           l.Logger
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "Use custom domain true",
			fields: fields{
				Config: config.NewThreeScale(config.ProductConfig{
					"CUSTOM_DOMAIN_ENABLED": "true",
				}),
				installation: &integreatlyv1alpha1.RHMI{
					Status: integreatlyv1alpha1.RHMIStatus{
						CustomDomain: &integreatlyv1alpha1.CustomDomainStatus{
							Enabled: true,
						}}},
			},
			want: true,
		},
		{
			name: "Don't use custom domain, normal flow",
			fields: fields{
				installation: getValidInstallation(integreatlyv1alpha1.InstallationTypeManagedApi),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				ConfigManager: tt.fields.ConfigManager,
				Config:        tt.fields.Config,
				mpm:           tt.fields.mpm,
				installation:  tt.fields.installation,
				tsClient:      tt.fields.tsClient,
				appsv1Client:  tt.fields.appsv1Client,
				oauthv1Client: tt.fields.oauthv1Client,
				Reconciler:    tt.fields.Reconciler,
				extraParams:   tt.fields.extraParams,
				recorder:      tt.fields.recorder,
				log:           tt.fields.log,
			}
			if got := r.useCustomDomain(); got != tt.want {
				t.Errorf("useCustomDomain() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconcileRatelimitPortAnnotation(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		ConfigManager config.ConfigReadWriter
		Config        *config.ThreeScale
		mpm           marketplace.MarketplaceInterface
		installation  *integreatlyv1alpha1.RHMI
		tsClient      ThreeScaleInterface
		appsv1Client  appsv1Client.AppsV1Interface
		oauthv1Client oauthClient.OauthV1Interface
		Reconciler    *resources.Reconciler
		extraParams   map[string]string
		recorder      record.EventRecorder
		log           l.Logger
	}

	type args struct {
		ctx          context.Context
		serverClient k8sclient.Client
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    integreatlyv1alpha1.StatusPhase
		wantErr bool
	}{
		{
			name: "No Annotation",
			fields: fields{
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": "test",
				}),
				installation: getTestInstallation("managed"),
			},
			args: args{
				ctx: context.TODO(),
				serverClient: fake.NewFakeClientWithScheme(scheme,
					&threescalev1.APIManager{
						ObjectMeta: metav1.ObjectMeta{
							Name:      apiManagerName,
							Namespace: "test",
						},
					}),
			},
			want:    integreatlyv1alpha1.PhaseCompleted,
			wantErr: false,
		},
		{
			name: "True Annotation",
			fields: fields{
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": "test",
				}),
				installation: getTestInstallation("managed"),
			},
			args: args{
				ctx: context.TODO(),
				serverClient: fake.NewFakeClientWithScheme(scheme,
					&threescalev1.APIManager{
						ObjectMeta: metav1.ObjectMeta{
							Name:        apiManagerName,
							Namespace:   "test",
							Annotations: map[string]string{"apps.3scale.net/disable-apicast-service-reconciler": "true"},
						},
					}),
			},
			want:    integreatlyv1alpha1.PhaseCompleted,
			wantErr: false,
		},
		{
			name: "False Annotation",
			fields: fields{
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": "test",
				}),
				installation: getTestInstallation("managed"),
			},
			args: args{
				ctx: context.TODO(),
				serverClient: fake.NewFakeClientWithScheme(scheme,
					&threescalev1.APIManager{
						ObjectMeta: metav1.ObjectMeta{
							Name:        apiManagerName,
							Namespace:   "test",
							Annotations: map[string]string{"apps.3scale.net/disable-apicast-service-reconciler": "false"},
						},
					}),
			},
			want:    integreatlyv1alpha1.PhaseCompleted,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				ConfigManager: tt.fields.ConfigManager,
				Config:        tt.fields.Config,
				mpm:           tt.fields.mpm,
				installation:  tt.fields.installation,
				tsClient:      tt.fields.tsClient,
				appsv1Client:  tt.fields.appsv1Client,
				oauthv1Client: tt.fields.oauthv1Client,
				Reconciler:    tt.fields.Reconciler,
				extraParams:   tt.fields.extraParams,
				recorder:      tt.fields.recorder,
				log:           tt.fields.log,
			}
			got, err := r.reconcileRatelimitPortAnnotation(tt.args.ctx, tt.args.serverClient)
			if (err != nil) != tt.wantErr {
				t.Errorf("reconcileComponents() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("reconcileComponents() got = %v, want %v", got, tt.want)
			}
		})
	}
}
func TestReconciler_reconcileExternalDatasources(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	postgres := &crov1.Postgres{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "threescale-postgres-rhmi",
			Namespace: "test",
		},
		Status: types.ResourceTypeStatus{
			Phase: types.PhaseComplete,
			SecretRef: &types.SecretRef{
				Name:      "test",
				Namespace: "test",
			},
		},
		Spec: types.ResourceTypeSpec{
			SecretRef: &types.SecretRef{
				Name:      "test",
				Namespace: "test",
			},
		},
	}
	backendRedis := &crov1.Redis{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "threescale-backend-redis-rhmi",
			Namespace: "test",
		},
		Status: types.ResourceTypeStatus{
			Phase: types.PhaseComplete,
			SecretRef: &types.SecretRef{
				Name:      "test",
				Namespace: "test",
			},
		},
		Spec: types.ResourceTypeSpec{
			SecretRef: &types.SecretRef{
				Name:      "test",
				Namespace: "test",
			},
		},
	}
	systemRedis := &crov1.Redis{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "threescale-redis-rhmi",
			Namespace: "test",
		},
		Status: types.ResourceTypeStatus{
			Phase: types.PhaseComplete,
			SecretRef: &types.SecretRef{
				Name:      "test",
				Namespace: "test",
			},
		},
		Spec: types.ResourceTypeSpec{
			SecretRef: &types.SecretRef{
				Name:      "test",
				Namespace: "test",
			},
		},
	}
	credSec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
	}
	redisSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "system-redis",
			Namespace: "test",
		},
		Data: map[string][]byte{
			"MESSAGE_BUS_URL":            []byte("Hello"),
			"MESSAGE_BUS_NAMESPACE":      []byte("Hello"),
			"MESSAGE_BUS_SENTINEL_HOSTS": []byte("Hello"),
			"MESSAGE_BUS_SENTINEL_ROLE":  []byte("Hello"),
		},
	}

	type fields struct {
		ConfigManager config.ConfigReadWriter
		Config        *config.ThreeScale
		mpm           marketplace.MarketplaceInterface
		installation  *integreatlyv1alpha1.RHMI
		tsClient      ThreeScaleInterface
		appsv1Client  appsv1Client.AppsV1Interface
		oauthv1Client oauthClient.OauthV1Interface
		Reconciler    *resources.Reconciler
	}
	type args struct {
		ctx          context.Context
		serverClient k8sclient.Client
	}
	tests := []struct {
		name                 string
		fields               fields
		args                 args
		want                 integreatlyv1alpha1.StatusPhase
		wantErr              bool
		verificationFunction func(k8sclient.Client) bool
	}{
		{
			name: "test initial install no MESSAGE_BUS keys",
			fields: fields{
				ConfigManager: nil,
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": "test",
				}),
				mpm:           nil,
				installation:  getTestInstallation("managed"),
				tsClient:      nil,
				appsv1Client:  nil,
				oauthv1Client: nil,
				Reconciler:    nil,
			},
			args: args{
				ctx:          context.TODO(),
				serverClient: fake.NewFakeClientWithScheme(scheme, postgres, backendRedis, systemRedis, credSec),
			},
			want:                 integreatlyv1alpha1.PhaseCompleted,
			wantErr:              false,
			verificationFunction: verifyMessageBusDoesNotExist,
		},
		{
			name: "test existing install no MESSAGE_BUS keys",
			fields: fields{
				ConfigManager: nil,
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": "test",
				}),
				mpm:           nil,
				installation:  getTestInstallation("managed"),
				tsClient:      nil,
				appsv1Client:  nil,
				oauthv1Client: nil,
				Reconciler:    nil,
			},
			args: args{
				ctx:          context.TODO(),
				serverClient: fake.NewFakeClientWithScheme(scheme, postgres, backendRedis, systemRedis, credSec, redisSecret),
			},
			want:                 integreatlyv1alpha1.PhaseCompleted,
			wantErr:              false,
			verificationFunction: verifyMessageBusDoesNotExist,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				ConfigManager: tt.fields.ConfigManager,
				Config:        tt.fields.Config,
				log:           getLogger(),
				mpm:           tt.fields.mpm,
				installation:  tt.fields.installation,
				tsClient:      tt.fields.tsClient,
				appsv1Client:  tt.fields.appsv1Client,
				oauthv1Client: tt.fields.oauthv1Client,
				Reconciler:    tt.fields.Reconciler,
			}
			got, err := r.reconcileExternalDatasources(tt.args.ctx, tt.args.serverClient)
			if (err != nil) != tt.wantErr {
				t.Errorf("reconcileExternalDatasources() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("reconcileExternalDatasources() got = %v, want %v", got, tt.want)
			}
			if !verifyMessageBusDoesNotExist(tt.args.serverClient) {
				t.Fatal("found message bus values in secret")
			}
		})
	}
}

func TestReconciler_getTenantAccountPassword(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	account := AccountDetail{
		Id:      1,
		Name:    "test-name",
		OrgName: "test-org-name",
	}

	type args struct {
		ctx            context.Context
		serverClient   k8sclient.Client
		shouldCheckPwd bool
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "tenant-account-passwords Secret doesn't exist",
			args: args{
				ctx:            context.TODO(),
				serverClient:   fake.NewFakeClientWithScheme(scheme),
				shouldCheckPwd: false,
			},
			want:    "",
			wantErr: false,
		},
		{
			name: "tenant-account-passwords Secret exists but is empty",
			args: args{
				ctx: context.TODO(),
				serverClient: fake.NewFakeClientWithScheme(scheme,
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "tenant-account-passwords",
							Namespace: "test-namespace",
						},
					}),
				shouldCheckPwd: false,
			},
			want:    "",
			wantErr: false,
		},
		{
			name: "tenant-account-passwords Secret exists but account doesn't have password yet",
			args: args{
				ctx: context.TODO(),
				serverClient: fake.NewFakeClientWithScheme(scheme,
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "tenant-account-passwords",
							Namespace: "test-namespace",
						},
						Data: map[string][]byte{
							"wrong-test-name": []byte("wrong-test-password"),
						},
					}),
				shouldCheckPwd: false,
			},
			want:    "",
			wantErr: false,
		},
		{
			name: "test tenant-account-passwords Secret exists and account has password",
			args: args{
				ctx: context.TODO(),
				serverClient: fake.NewFakeClientWithScheme(scheme,
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "tenant-account-passwords",
							Namespace: "test-namespace",
						},
						Data: map[string][]byte{
							"test-name": []byte("test-password"),
						},
					}),
				shouldCheckPwd: true,
			},
			want:    "test-password",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				ConfigManager: nil,
				Config:        config.NewThreeScale(config.ProductConfig{"NAMESPACE": "test-namespace"}),
				log:           getLogger(),
				mpm:           nil,
				installation:  getTestInstallation("multitenant-managed-api"),
				tsClient:      nil,
				appsv1Client:  nil,
				oauthv1Client: nil,
				Reconciler:    nil,
			}
			got, err := r.getTenantAccountPassword(tt.args.ctx, tt.args.serverClient, account)
			if (err != nil) != tt.wantErr {
				t.Errorf("getTenantAccountPassword() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.args.shouldCheckPwd {
				if got != tt.want {
					t.Errorf("getTenantAccountPassword() got = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestReconciler_removeTenantAccountPassword(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	account := AccountDetail{
		Id:      1,
		Name:    "test-name",
		OrgName: "test-org-name",
	}

	type args struct {
		ctx          context.Context
		serverClient k8sclient.Client
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "tenant-account-passwords Secret doesn't exist",
			args: args{
				ctx:          context.TODO(),
				serverClient: fake.NewFakeClientWithScheme(scheme),
			},
			wantErr: false,
		},
		{
			name: "tenant-account-passwords Secret exists but account doesn't have password",
			args: args{
				ctx: context.TODO(),
				serverClient: fake.NewFakeClientWithScheme(scheme,
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "tenant-account-passwords",
							Namespace: "test-namespace",
						},
					}),
			},
			wantErr: false,
		},
		{
			name: "tenant-account-passwords Secret exists and account has password",
			args: args{
				ctx: context.TODO(),
				serverClient: fake.NewFakeClientWithScheme(scheme,
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "tenant-account-passwords",
							Namespace: "test-namespace",
						},
						Data: map[string][]byte{
							"test-name": []byte("test-password"),
						},
					}),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				ConfigManager: nil,
				Config:        config.NewThreeScale(config.ProductConfig{"NAMESPACE": "test-namespace"}),
				log:           getLogger(),
				mpm:           nil,
				installation:  getTestInstallation("multitenant-managed-api"),
				tsClient:      nil,
				appsv1Client:  nil,
				oauthv1Client: nil,
				Reconciler:    nil,
			}
			err := r.removeTenantAccountPassword(tt.args.ctx, tt.args.serverClient, account)
			if (err != nil) != tt.wantErr {
				t.Errorf("removeTenantAccountPassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReconciler_getAccountsCreatedCM(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		ctx          context.Context
		serverClient k8sclient.Client
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "tenants-created ConfigMap doesn't exist",
			args: args{
				ctx:          context.TODO(),
				serverClient: fake.NewFakeClientWithScheme(scheme),
			},
			wantErr: false,
		},
		{
			name: "tenants-created ConfigMap exists",
			args: args{
				ctx: context.TODO(),
				serverClient: fake.NewFakeClientWithScheme(scheme,
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "tenants-created",
							Namespace: "test-namespace",
						},
					}),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := getAccountsCreatedCM(tt.args.ctx, tt.args.serverClient, "test-namespace")
			if (err != nil) != tt.wantErr {
				t.Errorf("getAccountsCreatedCM() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func verifyMessageBusDoesNotExist(serverClient k8sclient.Client) bool {
	redisSecret := &corev1.Secret{}
	err := serverClient.Get(context.TODO(), k8sclient.ObjectKey{Name: "system-redis", Namespace: "test"}, redisSecret)
	if err != nil {
		return false
	}
	messageBusKeys := []string{"MESSAGE_BUS_URL", "MESSAGE_BUS_NAMESPACE", "MESSAGE_BUS_SENTINEL_HOSTS", "MESSAGE_BUS_SENTINEL_ROLE"}
	for _, key := range messageBusKeys {
		if redisSecret.Data[key] != nil {
			return false
		}
	}
	return true
}

func TestReconciler_ping3scalePortals(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		ConfigManager config.ConfigReadWriter
		Config        *config.ThreeScale
		mpm           marketplace.MarketplaceInterface
		installation  *integreatlyv1alpha1.RHMI
		tsClient      ThreeScaleInterface
		appsv1Client  appsv1Client.AppsV1Interface
		oauthv1Client oauthClient.OauthV1Interface
		Reconciler    *resources.Reconciler
		extraParams   map[string]string
		recorder      record.EventRecorder
		log           l.Logger
	}
	type args struct {
		ctx          context.Context
		serverClient func() k8sclient.Client
		ips          []net.IP
	}

	systemSeed := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "system-seed",
			Namespace: "test",
		},
		Data: map[string][]byte{
			"MASTER_ACCESS_TOKEN": []byte("abc"),
		},
	}

	masterRoute := routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      labelRouteToSystemMaster,
			Namespace: "test",
			Labels: map[string]string{
				"zync.3scale.net/route-to": labelRouteToSystemMaster,
			},
		},
		Status: routev1.RouteStatus{
			Ingress: []routev1.RouteIngress{
				{
					Host: "host",
				},
			},
		},
	}

	developerRoute := routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      labelRouteToSystemDeveloper,
			Namespace: "test",
			Labels: map[string]string{
				"zync.3scale.net/route-to": labelRouteToSystemDeveloper,
			},
		},
		Status: routev1.RouteStatus{
			Ingress: []routev1.RouteIngress{
				{
					Host: "3scale-admin.example.com",
				},
			},
		},
	}

	providerRoute := routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      labelRouteToSystemProvider,
			Namespace: "test",
			Labels: map[string]string{
				"zync.3scale.net/route-to": labelRouteToSystemProvider,
			},
		},
		Status: routev1.RouteStatus{
			Ingress: []routev1.RouteIngress{
				{
					Host: "3scale.example.com",
				},
			},
		},
	}

	tests := []struct {
		name       string
		fields     fields
		args       args
		want       integreatlyv1alpha1.StatusPhase
		wantErr    bool
		errMessage string
	}{
		{
			name: "Get Master Token Failed",
			fields: fields{
				installation: &integreatlyv1alpha1.RHMI{
					Status: integreatlyv1alpha1.RHMIStatus{},
				},
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": "test",
				}),
			},
			args: args{
				ctx: context.TODO(),
				serverClient: func() k8sclient.Client {
					mockClient := moqclient.NewSigsClientMoqWithScheme(scheme)
					return mockClient
				},
			},
			want:       integreatlyv1alpha1.PhaseFailed,
			wantErr:    true,
			errMessage: "secrets \"system-seed\" not found",
		},
		{
			name: "List tenant accounts failed",
			fields: fields{
				installation: &integreatlyv1alpha1.RHMI{
					Status: integreatlyv1alpha1.RHMIStatus{},
				},
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": "test",
				}),
				tsClient: &ThreeScaleInterfaceMock{
					ListTenantAccountsFunc: func(accessToken string, page int, filterFn func(ac AccountDetail) bool) ([]AccountDetail, error) {
						return nil, errors.New("test no accounts returned")
					},
				},
			},
			args: args{
				ctx: context.TODO(),
				serverClient: func() k8sclient.Client {
					mockClient := moqclient.NewSigsClientMoqWithScheme(scheme, systemSeed)
					return mockClient
				},
			},
			want:       integreatlyv1alpha1.PhaseFailed,
			wantErr:    true,
			errMessage: "test no accounts returned",
		},
		{
			name: "Get system master route failed",
			fields: fields{
				installation: &integreatlyv1alpha1.RHMI{
					Status: integreatlyv1alpha1.RHMIStatus{},
				},
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": "test",
				}),
				tsClient: &ThreeScaleInterfaceMock{
					ListTenantAccountsFunc: func(accessToken string, page int, filterFn func(ac AccountDetail) bool) ([]AccountDetail, error) {
						return nil, nil
					},
				},
			},
			args: args{
				ctx: context.TODO(),
				serverClient: func() k8sclient.Client {
					mockClient := moqclient.NewSigsClientMoqWithScheme(scheme, systemSeed)
					mockClient.ListFunc = func(ctx context.Context, list k8sclient.ObjectList, opts ...k8sclient.ListOption) error {
						return errors.New("failed to list routes")
					}
					return mockClient
				},
			},
			want:       integreatlyv1alpha1.PhaseFailed,
			wantErr:    true,
			errMessage: "failed to retrieve system-master 3scale route",
		},
		{
			name: "Get system developer route failed",
			fields: fields{
				installation: &integreatlyv1alpha1.RHMI{
					Status: integreatlyv1alpha1.RHMIStatus{},
				},
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": "test",
				}),
				tsClient: &ThreeScaleInterfaceMock{
					ListTenantAccountsFunc: func(accessToken string, page int, filterFn func(ac AccountDetail) bool) ([]AccountDetail, error) {
						return nil, nil
					},
				},
			},
			args: args{
				ctx: context.TODO(),
				serverClient: func() k8sclient.Client {
					mockClient := moqclient.NewSigsClientMoqWithScheme(scheme, systemSeed)
					mockClient.ListFunc = func(ctx context.Context, list k8sclient.ObjectList, opts ...k8sclient.ListOption) error {
						listOpts := k8sclient.ListOptions{}
						listOpts.ApplyOptions(opts)

						if listOpts.LabelSelector.Matches(labels.Set(map[string]string{"zync.3scale.net/route-to": labelRouteToSystemMaster})) {
							list = &routev1.RouteList{ // nolint:ineffassign
								Items: []routev1.Route{
									masterRoute,
								},
							}
							return nil
						}
						return errors.New("failed to list routes")
					}
					return mockClient
				},
			},
			want:       integreatlyv1alpha1.PhaseFailed,
			wantErr:    true,
			errMessage: "failed to retrieve system-developer 3scale route",
		},
		{
			name: "Get system Provider route failed",
			fields: fields{
				installation: &integreatlyv1alpha1.RHMI{
					Status: integreatlyv1alpha1.RHMIStatus{},
				},
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": "test",
				}),
				tsClient: &ThreeScaleInterfaceMock{
					ListTenantAccountsFunc: func(accessToken string, page int, filterFn func(ac AccountDetail) bool) ([]AccountDetail, error) {
						return nil, nil
					},
				},
			},
			args: args{
				ctx: context.TODO(),
				serverClient: func() k8sclient.Client {
					mockClient := moqclient.NewSigsClientMoqWithScheme(scheme, systemSeed)
					mockClient.ListFunc = func(ctx context.Context, list k8sclient.ObjectList, opts ...k8sclient.ListOption) error {
						listOpts := k8sclient.ListOptions{}
						listOpts.ApplyOptions(opts)

						if listOpts.LabelSelector.Matches(labels.Set(map[string]string{"zync.3scale.net/route-to": labelRouteToSystemMaster})) {
							list = &routev1.RouteList{ // nolint:ineffassign
								Items: []routev1.Route{
									masterRoute,
								},
							}
							return nil
						}
						if listOpts.LabelSelector.Matches(labels.Set(map[string]string{"zync.3scale.net/route-to": labelRouteToSystemDeveloper})) {
							list = &routev1.RouteList{ // nolint:ineffassign
								Items: []routev1.Route{
									masterRoute,
									developerRoute,
								},
							}
							return nil
						}

						return errors.New("failed to list routes")
					}
					return mockClient
				},
			},
			want:       integreatlyv1alpha1.PhaseFailed,
			wantErr:    true,
			errMessage: "failed to retrieve system-provider 3scale route",
		},
		{
			name: "failed to ping 3scale portal",
			fields: fields{
				installation: &integreatlyv1alpha1.RHMI{
					Status: integreatlyv1alpha1.RHMIStatus{},
				},
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": "test",
				}),
				tsClient: &ThreeScaleInterfaceMock{
					ListTenantAccountsFunc: func(accessToken string, page int, filterFn func(ac AccountDetail) bool) ([]AccountDetail, error) {
						return []AccountDetail{
							{
								AdminBaseURL: "3scale-admin.example.com",
								State:        "approved",
							},
						}, nil
					},
				},
			},
			args: args{
				ctx: context.TODO(),
				serverClient: func() k8sclient.Client {
					mockClient := moqclient.NewSigsClientMoqWithScheme(scheme, systemSeed,
						&masterRoute,
						&developerRoute,
						&providerRoute,
					)
					return mockClient
				},
				ips: []net.IP{
					{127, 0, 0, 1},
				},
			},
			want:       integreatlyv1alpha1.PhaseFailed,
			wantErr:    true,
			errMessage: "failed to ping",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				ConfigManager: tt.fields.ConfigManager,
				Config:        tt.fields.Config,
				mpm:           tt.fields.mpm,
				installation:  tt.fields.installation,
				tsClient:      tt.fields.tsClient,
				appsv1Client:  tt.fields.appsv1Client,
				oauthv1Client: tt.fields.oauthv1Client,
				Reconciler:    tt.fields.Reconciler,
				extraParams:   tt.fields.extraParams,
				recorder:      tt.fields.recorder,
				log:           tt.fields.log,
			}
			got, err := r.ping3scalePortals(tt.args.ctx, tt.args.serverClient(), tt.args.ips)
			if (err != nil) != tt.wantErr {
				t.Errorf("ping3scalePortals() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errMessage) {
				t.Errorf("Unexpected error message returned: \nRecived error: %s\nExpected error to contain: %s", err, tt.errMessage)
			}
			if got != tt.want {
				t.Errorf("ping3scalePortals() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_reconcileCustomDomainAlerts(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		ConfigManager config.ConfigReadWriter
		Config        *config.ThreeScale
		mpm           marketplace.MarketplaceInterface
		installation  *integreatlyv1alpha1.RHMI
		tsClient      ThreeScaleInterface
		appsv1Client  appsv1Client.AppsV1Interface
		oauthv1Client oauthClient.OauthV1Interface
		Reconciler    *resources.Reconciler
		extraParams   map[string]string
		recorder      record.EventRecorder
		log           l.Logger
	}
	type args struct {
		ctx    context.Context
		client k8sclient.Client
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    integreatlyv1alpha1.StatusPhase
		wantErr bool
	}{
		{
			name: "Alerts are created",
			args: args{
				ctx:    context.TODO(),
				client: moqclient.NewSigsClientMoqWithScheme(scheme),
			},
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{
					ReadObservabilityFunc: func() (*config.Observability, error) {
						return &config.Observability{
							Config: config.ProductConfig{
								"NAMESPACE": "namespace",
							},
						}, nil
					}},
				installation: &integreatlyv1alpha1.RHMI{
					Spec: integreatlyv1alpha1.RHMISpec{Type: "managed-api"},
					Status: integreatlyv1alpha1.RHMIStatus{
						CustomDomain: &integreatlyv1alpha1.CustomDomainStatus{Error: "Some error"},
					},
				},
				log: getLogger(),
			},
			want:    integreatlyv1alpha1.PhaseCompleted,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				ConfigManager: tt.fields.ConfigManager,
				Config:        tt.fields.Config,
				mpm:           tt.fields.mpm,
				installation:  tt.fields.installation,
				tsClient:      tt.fields.tsClient,
				appsv1Client:  tt.fields.appsv1Client,
				oauthv1Client: tt.fields.oauthv1Client,
				Reconciler:    tt.fields.Reconciler,
				extraParams:   tt.fields.extraParams,
				recorder:      tt.fields.recorder,
				log:           tt.fields.log,
			}
			got, err := r.reconcileCustomDomainAlerts(tt.args.ctx, tt.args.client)
			if (err != nil) != tt.wantErr {
				t.Errorf("reconcileCustomDomainAlerts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("reconcileCustomDomainAlerts() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_checkRedirects(t *testing.T) {
	type args struct {
		host       string
		path       string
		res        *http.Response
		statusCode int
	}
	tests := []struct {
		name       string
		args       args
		want       bool
		statusCode int
	}{
		{
			name:       "Found nested expected response",
			want:       true,
			statusCode: 302,
			args: args{
				host:       "example.com",
				path:       "/p/login",
				statusCode: 302,
				res: &http.Response{
					StatusCode: 400,
					Request: &http.Request{
						URL: &url.URL{
							Host: "example.com",
							Path: "/bad/path",
						},
						Response: &http.Response{
							StatusCode: 302,
							Request: &http.Request{
								URL: &url.URL{
									Host: "example.com",
									Path: "/bad/path/again",
								},
								Response: &http.Response{
									StatusCode: 302,
									Request: &http.Request{
										URL: &url.URL{
											Host: "example.com",
											Path: "/p/login",
										},
										Response: &http.Response{
											StatusCode: 301,
											Request: &http.Request{
												URL: &url.URL{
													Host: "example.com",
													Path: "",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:       "Nil point for request paced",
			want:       false,
			statusCode: 000,
			args: args{
				host:       "example.com",
				path:       "/p/login",
				statusCode: 302,
				res:        nil,
			},
		},
		{
			name:       "Did not find expected response: Missing status code",
			want:       false,
			statusCode: 000,
			args: args{
				host:       "example.com",
				path:       "/p/login",
				statusCode: 302,
				res: &http.Response{
					StatusCode: 400,
					Request: &http.Request{
						URL: &url.URL{
							Host: "example.com",
							Path: "/p/login",
						},
						Response: &http.Response{
							StatusCode: 200,
							Request: &http.Request{
								URL: &url.URL{
									Host: "example.com",
									Path: "/p/login",
								},
							},
						},
					},
				},
			},
		},

		{
			name:       "Did not find expected response: Missing host",
			want:       false,
			statusCode: 000,
			args: args{
				host:       "example.com",
				path:       "/p/login",
				statusCode: 302,
				res: &http.Response{
					StatusCode: 400,
					Request: &http.Request{
						URL: &url.URL{
							Host: "example.com",
							Path: "/bad/path",
						},
						Response: &http.Response{
							StatusCode: 302,
							Request: &http.Request{
								URL: &url.URL{
									Host: "example.com",
									Path: "/bad/path/again",
								},
								Response: &http.Response{
									StatusCode: 302,
									Request: &http.Request{
										URL: &url.URL{
											Host: "wrong.example.com",
											Path: "/p/login",
										},
										Response: &http.Response{
											StatusCode: 301,
											Request: &http.Request{
												URL: &url.URL{
													Host: "example.com",
													Path: "",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},

		{
			name:       "Did not find expected response: Missing path",
			want:       false,
			statusCode: 000,
			args: args{
				host:       "example.com",
				path:       "/p/login",
				statusCode: 302,
				res: &http.Response{
					StatusCode: 400,
					Request: &http.Request{
						URL: &url.URL{
							Host: "example.com",
							Path: "/bad/path",
						},
						Response: &http.Response{
							StatusCode: 302,
							Request: &http.Request{
								URL: &url.URL{
									Host: "example.com",
									Path: "/bad/path/again",
								},
								Response: &http.Response{
									StatusCode: 302,
									Request: &http.Request{
										URL: &url.URL{
											Host: "example.com",
											Path: "/p/login/path/m",
										},
										Response: &http.Response{
											StatusCode: 301,
											Request: &http.Request{
												URL: &url.URL{
													Host: "example.com",
													Path: "",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, statusCode := checkRedirects(tt.args.host, tt.args.path, tt.args.res, tt.args.statusCode)
			if got != tt.want {
				t.Errorf("checkRedirects() got = %v, want %v", got, tt.want)
			}
			if statusCode != tt.statusCode {
				t.Errorf("checkRedirects() statusCode = %v, want %v", statusCode, tt.statusCode)
			}
		})
	}
}

func mockDNS(host, ip string) (*mockdns.Server, error) {
	srv, err := mockdns.NewServer(map[string]mockdns.Zone{
		fmt.Sprintf("%s.", host): {
			A: []string{ip},
		},
		fmt.Sprintf("%s:443.", host): {
			A: []string{ip},
		},
	}, false)
	if err != nil {
		return nil, err
	}
	srv.PatchNet(net.DefaultResolver)
	return srv, nil
}

func mockHTTP(ip string) (*httptest.Server, error) {
	// create a listener with the desired port
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:10620", ip))
	if err != nil {
		return nil, err
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case fmt.Sprintf("/%s", labelRouteToSystemProvider):
			w.WriteHeader(http.StatusOK)
			return
		case fmt.Sprintf("/%s", labelRouteToSystemDeveloper):
			w.WriteHeader(http.StatusOK)
			return
		case fmt.Sprintf("/%s", labelRouteToSystemMaster):
			w.WriteHeader(http.StatusOK)
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	})
	srv := httptest.NewUnstartedServer(handler)
	// NewUnstartedServer creates a listener. Close that listener and replace with the one we created
	srv.Listener.Close()
	srv.Listener = listener
	srv.StartTLS()
	return srv, nil
}

func productConfigMock() *quota.ProductConfigMock {
	return &quota.ProductConfigMock{
		ConfigureFunc: func(obj metav1.Object) error {
			return nil
		},
		GetActiveQuotaFunc:     nil,
		GetRateLimitConfigFunc: nil,
		GetReplicasFunc:        nil,
		GetResourceConfigFunc:  nil,
	}
}

func TestReconciler_addSSOReadyAnnotationToUser(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	testUser := &usersv1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-user01",
		},
	}

	type fields struct {
		ConfigManager config.ConfigReadWriter
		Config        *config.ThreeScale
		mpm           marketplace.MarketplaceInterface
		installation  *integreatlyv1alpha1.RHMI
		tsClient      ThreeScaleInterface
		appsv1Client  appsv1Client.AppsV1Interface
		oauthv1Client oauthClient.OauthV1Interface
		Reconciler    *resources.Reconciler
		extraParams   map[string]string
		recorder      record.EventRecorder
		log           l.Logger
	}
	type args struct {
		client   k8sclient.Client
		userName string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Success on valid User CR",
			fields: fields{
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": defaultInstallationNamespace,
				}),
				log: getLogger(),
			},
			args: args{
				client:   fake.NewFakeClientWithScheme(scheme, testUser),
				userName: "test-user01",
			},
			wantErr: false,
		},
		{
			name: "Fail on non-existent User CR",
			fields: fields{
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": defaultInstallationNamespace,
				}),
				log: getLogger(),
			},
			args: args{
				client:   fake.NewFakeClientWithScheme(scheme, testUser),
				userName: "bad-username",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				ConfigManager: tt.fields.ConfigManager,
				Config:        tt.fields.Config,
				mpm:           tt.fields.mpm,
				installation:  tt.fields.installation,
				tsClient:      tt.fields.tsClient,
				appsv1Client:  tt.fields.appsv1Client,
				oauthv1Client: tt.fields.oauthv1Client,
				Reconciler:    tt.fields.Reconciler,
				extraParams:   tt.fields.extraParams,
				recorder:      tt.fields.recorder,
				log:           tt.fields.log,
			}
			if err := r.addSSOReadyAnnotationToUser(context.TODO(), tt.args.client, tt.args.userName); (err != nil) != tt.wantErr {
				t.Errorf("addSSOReadyAnnotationToUser() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReconciler_SetAdminDetailsOnSecret(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "system-seed",
			Namespace: defaultInstallationNamespace,
		},
		Data: map[string][]byte{
			"ADMIN_USER":  []byte(""),
			"ADMIN_EMAIL": []byte(""),
		},
	}

	type fields struct {
		ConfigManager config.ConfigReadWriter
		Config        *config.ThreeScale
		mpm           marketplace.MarketplaceInterface
		installation  *integreatlyv1alpha1.RHMI
		tsClient      ThreeScaleInterface
		appsv1Client  appsv1Client.AppsV1Interface
		oauthv1Client oauthClient.OauthV1Interface
		Reconciler    *resources.Reconciler
		extraParams   map[string]string
		recorder      record.EventRecorder
		log           l.Logger
	}
	type args struct {
		serverClient k8sclient.Client
		username     string
		email        string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Success when secret exists",
			fields: fields{
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": defaultInstallationNamespace,
				}),
				log: getLogger(),
			},
			args: args{
				serverClient: fake.NewFakeClientWithScheme(scheme, secret),
				username:     "username",
				email:        "email",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				ConfigManager: tt.fields.ConfigManager,
				Config:        tt.fields.Config,
				mpm:           tt.fields.mpm,
				installation:  tt.fields.installation,
				tsClient:      tt.fields.tsClient,
				appsv1Client:  tt.fields.appsv1Client,
				oauthv1Client: tt.fields.oauthv1Client,
				Reconciler:    tt.fields.Reconciler,
				extraParams:   tt.fields.extraParams,
				recorder:      tt.fields.recorder,
				log:           tt.fields.log,
			}
			if err := r.SetAdminDetailsOnSecret(context.TODO(), tt.args.serverClient, tt.args.username, tt.args.email); (err != nil) != tt.wantErr {
				t.Errorf("SetAdminDetailsOnSecret() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReconciler_createStsS3Secret(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		ConfigManager config.ConfigReadWriter
		Config        *config.ThreeScale
		mpm           marketplace.MarketplaceInterface
		installation  *integreatlyv1alpha1.RHMI
		tsClient      ThreeScaleInterface
		appsv1Client  appsv1Client.AppsV1Interface
		oauthv1Client oauthClient.OauthV1Interface
		Reconciler    *resources.Reconciler
		extraParams   map[string]string
		recorder      record.EventRecorder
		log           l.Logger
	}
	type args struct {
		ctx            context.Context
		serverClient   k8sclient.Client
		credSec        *corev1.Secret
		blobStorageSec *corev1.Secret
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test unable to get secret",
			args: args{
				ctx:            context.TODO(),
				serverClient:   fake.NewClientBuilder().WithScheme(scheme).Build(),
				credSec:        &corev1.Secret{},
				blobStorageSec: &corev1.Secret{},
			},
			fields: fields{
				Config: config.NewThreeScale(config.ProductConfig{}),
			},
			wantErr: true,
		},
		{
			name: "test success creating s3 secret",
			args: args{
				ctx: context.TODO(),
				serverClient: fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      stsS3CredentialsSecretName,
						Namespace: defaultInstallationNamespace,
					},
					Data: map[string][]byte{
						"role_arn": []byte("roleArn"),
					},
				}).Build(),
				credSec: &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      s3CredentialsSecretName,
						Namespace: defaultInstallationNamespace,
					},
					Data: map[string][]byte{},
				},
				blobStorageSec: &corev1.Secret{
					Data: map[string][]byte{
						threescaleAmp.AwsBucket: []byte("bucket"),
						threescaleAmp.AwsRegion: []byte("region"),
					},
				},
			},
			fields: fields{
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": defaultInstallationNamespace,
				}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				ConfigManager: tt.fields.ConfigManager,
				Config:        tt.fields.Config,
				mpm:           tt.fields.mpm,
				installation:  tt.fields.installation,
				tsClient:      tt.fields.tsClient,
				appsv1Client:  tt.fields.appsv1Client,
				oauthv1Client: tt.fields.oauthv1Client,
				Reconciler:    tt.fields.Reconciler,
				extraParams:   tt.fields.extraParams,
				recorder:      tt.fields.recorder,
				log:           tt.fields.log,
			}
			if err := r.createStsS3Secret(tt.args.ctx, tt.args.serverClient, tt.args.credSec, tt.args.blobStorageSec); (err != nil) != tt.wantErr {
				t.Errorf("createStsS3Secret() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReconciler_createMCGS3Secret(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	obc := &noobaav1.ObjectBucketClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mcg.ThreescaleBucketClaim,
			Namespace: mcg.DefaultInstallationNamespace + "-operator",
		},
		Spec: obv1.ObjectBucketClaimSpec{
			BucketName: mcg.ThreescaleBucketClaim,
		},
		Status: obv1.ObjectBucketClaimStatus{
			Phase: obv1.ObjectBucketClaimStatusPhaseBound,
		},
	}

	bucketSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mcg.ThreescaleBucketClaim,
			Namespace: mcg.DefaultInstallationNamespace + "-operator",
		},
		Data: map[string][]byte{
			threescaleAmp.AwsAccessKeyID:     []byte("id"),
			threescaleAmp.AwsSecretAccessKey: []byte("key"),
		},
	}

	s3Route := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s3RouteName,
			Namespace: mcg.DefaultInstallationNamespace + "-operator",
		},
	}

	type fields struct {
		ConfigManager config.ConfigReadWriter
		Config        *config.ThreeScale
		mpm           marketplace.MarketplaceInterface
		installation  *integreatlyv1alpha1.RHMI
		tsClient      ThreeScaleInterface
		appsv1Client  appsv1Client.AppsV1Interface
		oauthv1Client oauthClient.OauthV1Interface
		Reconciler    *resources.Reconciler
		extraParams   map[string]string
		recorder      record.EventRecorder
		log           l.Logger
	}
	type args struct {
		ctx          context.Context
		serverClient k8sclient.Client
		credSec      *corev1.Secret
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test unable to get object bucket claim",
			args: args{
				ctx:          context.TODO(),
				serverClient: moqclient.NewSigsClientMoqWithScheme(scheme),
				credSec:      &corev1.Secret{},
			},
			fields: fields{
				Config:       config.NewThreeScale(config.ProductConfig{}),
				installation: &integreatlyv1alpha1.RHMI{},
			},
			wantErr: true,
		},
		{
			name: "test unable to get secret",
			args: args{
				ctx:          context.TODO(),
				serverClient: moqclient.NewSigsClientMoqWithScheme(scheme, obc),
				credSec:      &corev1.Secret{},
			},
			fields: fields{
				Config:       config.NewThreeScale(config.ProductConfig{}),
				installation: &integreatlyv1alpha1.RHMI{},
			},
			wantErr: true,
		},
		{
			name: "test unable to get route",
			args: args{
				ctx:          context.TODO(),
				serverClient: moqclient.NewSigsClientMoqWithScheme(scheme, obc, bucketSecret),
				credSec:      &corev1.Secret{},
			},
			fields: fields{
				Config:       config.NewThreeScale(config.ProductConfig{}),
				installation: &integreatlyv1alpha1.RHMI{},
			},
			wantErr: true,
		},
		{
			name: "test successfully create s3 credentials for mcg",
			args: args{
				ctx:          context.TODO(),
				serverClient: moqclient.NewSigsClientMoqWithScheme(scheme, obc, bucketSecret, s3Route),
				credSec: &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      s3CredentialsSecretName,
						Namespace: defaultInstallationNamespace,
					},
					Data: map[string][]byte{},
				},
			},
			fields: fields{
				Config:       config.NewThreeScale(config.ProductConfig{}),
				installation: &integreatlyv1alpha1.RHMI{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				ConfigManager: tt.fields.ConfigManager,
				Config:        tt.fields.Config,
				mpm:           tt.fields.mpm,
				installation:  tt.fields.installation,
				tsClient:      tt.fields.tsClient,
				appsv1Client:  tt.fields.appsv1Client,
				oauthv1Client: tt.fields.oauthv1Client,
				Reconciler:    tt.fields.Reconciler,
				extraParams:   tt.fields.extraParams,
				recorder:      tt.fields.recorder,
				log:           tt.fields.log,
			}
			if err := r.createMCGS3Secret(tt.args.ctx, tt.args.serverClient, tt.args.credSec); (err != nil) != tt.wantErr {
				t.Errorf("createMCGS3Secret() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
