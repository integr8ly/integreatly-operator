package controllers

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	crov1alpha1 "github.com/integr8ly/cloud-resource-operator/api/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/utils"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	controllerruntime "sigs.k8s.io/controller-runtime"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	"github.com/integr8ly/integreatly-operator/internal/controller/subscription/csvlocator"

	catalogsourceClient "github.com/integr8ly/integreatly-operator/pkg/resources/catalogsource"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	operatorNamespace      = "openshift-operators"
	defaultInstallPlanName = "installplan"
)

func TestSubscriptionReconciler(t *testing.T) {

	csv := &operatorsv1alpha1.ClusterServiceVersion{
		Spec: operatorsv1alpha1.ClusterServiceVersionSpec{
			Replaces: "123",
		},
	}
	csvStringfied, err := json.Marshal(csv)
	if err != nil {
		panic(err)
	}

	installPlan := &operatorsv1alpha1.InstallPlan{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultInstallPlanName,
			Namespace: operatorNamespace,
		},
		Status: operatorsv1alpha1.InstallPlanStatus{
			Plan: []*operatorsv1alpha1.Step{
				{
					Resource: operatorsv1alpha1.StepResource{
						Kind:     operatorsv1alpha1.ClusterServiceVersionKind,
						Manifest: string(csvStringfied),
					},
				},
			},
		},
	}

	rhmiCR := &integreatlyv1alpha1.RHMI{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rhmi",
			Namespace: operatorNamespace,
		},
	}

	scenarios := []struct {
		Name                string
		Request             reconcile.Request
		APISubscription     *operatorsv1alpha1.Subscription
		catalogsourceClient catalogsourceClient.CatalogSourceClientInterface
		Verify              func(client k8sclient.Client, res reconcile.Result, err error, t *testing.T)
	}{
		{
			Name: "subscription controller changes integreatly Subscription from automatic to manual",
			Request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: operatorNamespace,
					Name:      IntegreatlyPackage,
				},
			},
			APISubscription: &operatorsv1alpha1.Subscription{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: operatorNamespace,
					Name:      IntegreatlyPackage,
				},
				Spec: &operatorsv1alpha1.SubscriptionSpec{
					InstallPlanApproval: operatorsv1alpha1.ApprovalAutomatic,
				},
				Status: operatorsv1alpha1.SubscriptionStatus{
					InstallPlanRef: &corev1.ObjectReference{
						Name:      installPlan.Name,
						Namespace: installPlan.Namespace,
					},
				},
			},
			Verify: func(c k8sclient.Client, res reconcile.Result, err error, t *testing.T) {
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}
				sub := &operatorsv1alpha1.Subscription{}
				err = c.Get(context.TODO(), k8sclient.ObjectKey{Name: IntegreatlyPackage, Namespace: operatorNamespace}, sub)
				if err != nil {
					t.Fatalf("unexpected error getting subscription: %s", err.Error())
				}
				if sub.Spec.InstallPlanApproval != operatorsv1alpha1.ApprovalManual {
					t.Fatalf("expected Manual but got %s", sub.Spec.InstallPlanApproval)
				}
			},
			catalogsourceClient: getCatalogSourceClient(""),
		},
		{
			Name: "subscription controller doesn't change subscription in different namespace",
			Request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "other-ns",
					Name:      IntegreatlyPackage,
				},
			},
			APISubscription: &operatorsv1alpha1.Subscription{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "other-ns",
					Name:      IntegreatlyPackage,
				},
				Spec: &operatorsv1alpha1.SubscriptionSpec{
					InstallPlanApproval: operatorsv1alpha1.ApprovalAutomatic,
				},
				Status: operatorsv1alpha1.SubscriptionStatus{
					InstallPlanRef: &corev1.ObjectReference{
						Name:      installPlan.Name,
						Namespace: installPlan.Namespace,
					},
				},
			},
			Verify: func(c k8sclient.Client, res reconcile.Result, err error, t *testing.T) {
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}
				sub := &operatorsv1alpha1.Subscription{}
				err = c.Get(context.TODO(), k8sclient.ObjectKey{Name: IntegreatlyPackage, Namespace: "other-ns"}, sub)
				if err != nil {
					t.Fatalf("unexpected error getting subscription : %s", err.Error())
				}
				if sub.Spec.InstallPlanApproval != operatorsv1alpha1.ApprovalAutomatic {
					t.Fatalf("expected Automatic but got %s", sub.Spec.InstallPlanApproval)
				}
			},
			catalogsourceClient: getCatalogSourceClient(""),
		},
		{
			Name: "subscription controller doesn't change other subscription in the same namespace",
			Request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: operatorNamespace,
					Name:      "other-package",
				},
			},
			APISubscription: &operatorsv1alpha1.Subscription{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: operatorNamespace,
					Name:      "other-package",
				},
				Spec: &operatorsv1alpha1.SubscriptionSpec{
					InstallPlanApproval: operatorsv1alpha1.ApprovalAutomatic,
				},
				Status: operatorsv1alpha1.SubscriptionStatus{
					InstallPlanRef: &corev1.ObjectReference{
						Name:      installPlan.Name,
						Namespace: installPlan.Namespace,
					},
				},
			},
			Verify: func(c k8sclient.Client, res reconcile.Result, err error, t *testing.T) {
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}
				sub := &operatorsv1alpha1.Subscription{}
				err = c.Get(context.TODO(), k8sclient.ObjectKey{Name: "other-package", Namespace: operatorNamespace}, sub)
				if err != nil {
					t.Fatalf("unexpected error getting subscription: %s", err.Error())
				}
				if sub.Spec.InstallPlanApproval != operatorsv1alpha1.ApprovalAutomatic {
					t.Fatalf("expected Automatic but got %s", sub.Spec.InstallPlanApproval)
				}
			},
			catalogsourceClient: getCatalogSourceClient(""),
		},
		{
			Name: "subscription controller handles when subscription is missing",
			Request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: operatorNamespace,
					Name:      IntegreatlyPackage,
				},
			},
			APISubscription: &operatorsv1alpha1.Subscription{},
			Verify: func(c k8sclient.Client, res reconcile.Result, err error, t *testing.T) {
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}
			},
			catalogsourceClient: getCatalogSourceClient(""),
		},
		{
			Name: "subscription controller changes the subscription status block to trigger the recreation of a installplan",
			Request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: operatorNamespace,
					Name:      IntegreatlyPackage,
				},
			},
			APISubscription: &operatorsv1alpha1.Subscription{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: operatorNamespace,
					Name:      IntegreatlyPackage,
				},
				Spec: &operatorsv1alpha1.SubscriptionSpec{
					InstallPlanApproval: operatorsv1alpha1.ApprovalManual,
				},
				Status: operatorsv1alpha1.SubscriptionStatus{
					InstallPlanRef: &corev1.ObjectReference{
						Name:      installPlan.Name,
						Namespace: installPlan.Namespace,
					},
					InstalledCSV: "123",
					CurrentCSV:   "124",
				},
			},
			Verify: func(c k8sclient.Client, res reconcile.Result, err error, t *testing.T) {
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}

				sub := &operatorsv1alpha1.Subscription{}
				err = c.Get(context.TODO(), k8sclient.ObjectKey{Name: IntegreatlyPackage, Namespace: operatorNamespace}, sub)
				if err != nil {
					t.Fatalf("unexpected error getting sublscription: %s", err.Error())
				}
				if res.RequeueAfter == 0 {
					t.Fatalf("expected reconciler to await manual approval of the upgrade")
				}
			},
			catalogsourceClient: getCatalogSourceClient(""),
		},
	}

	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			APIObject := scenario.APISubscription
			client := utils.NewTestClient(scheme, APIObject, installPlan, rhmiCR)
			reconciler := SubscriptionReconciler{
				Client:              client,
				Scheme:              scheme,
				catalogSourceClient: scenario.catalogsourceClient,
				operatorNamespace:   operatorNamespace,
				csvLocator:          &csvlocator.EmbeddedCSVLocator{},
			}
			res, err := reconciler.Reconcile(context.TODO(), scenario.Request)
			scenario.Verify(client, res, err, t)
		})
	}
}

func TestShouldReconcileSubscription(t *testing.T) {
	scenarios := []struct {
		Name           string
		Namespace      string
		Request        reconcile.Request
		ExpectedResult bool
	}{
		{
			Name:      "Non matching namespace",
			Namespace: "testing-namespaces-operator",
			Request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "integreatly",
					Namespace: "another",
				},
			},
			ExpectedResult: false,
		},
		{
			Name:      "Not in reconcile name list",
			Namespace: "testing-namespaces-operator",
			Request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "another",
					Namespace: "testing-namespaces-operator",
				},
			},
			ExpectedResult: false,
		},
		{
			Name:      "\"integreatly\" subscription",
			Namespace: "testing-namespaces-operator",
			Request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "integreatly",
					Namespace: "testing-namespaces-operator",
				},
			},
			ExpectedResult: true,
		},
		{
			Name:      "Managed API Addon subscription",
			Namespace: "testing-namespaces-operator",
			Request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "addon-managed-api-service",
					Namespace: "testing-namespaces-operator",
				},
			},
			ExpectedResult: true,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			reconciler := &SubscriptionReconciler{
				operatorNamespace: scenario.Namespace,
			}

			result := reconciler.shouldReconcileSubscription(scenario.Request)

			if result != scenario.ExpectedResult {
				t.Errorf("Unexpected result. Expected %v, got %v", scenario.ExpectedResult, result)
			}
		})
	}
}

func getCatalogSourceClient(replaces string) catalogsourceClient.CatalogSourceClientInterface {
	return &catalogsourceClient.CatalogSourceClientInterfaceMock{
		GetLatestCSVFunc: func(catalogSourceKey types.NamespacedName, packageName, channelName string) (*operatorsv1alpha1.ClusterServiceVersion, error) {
			return &operatorsv1alpha1.ClusterServiceVersion{
				Spec: operatorsv1alpha1.ClusterServiceVersionSpec{
					Replaces: replaces,
				},
			}, nil
		},
	}
}

func TestAllowDatabaseUpdates(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		Client k8sclient.Client
	}

	scenarios := []struct {
		Name                   string
		RHMI                   integreatlyv1alpha1.RHMI
		Fields                 fields
		IsServiceAffecting     bool
		ExpectedUpdatesAllowed bool
	}{
		{
			Name: "updates allowed when Version is not empty, toVersion is not empty and upgrade is service affecting",
			RHMI: integreatlyv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testrhmi",
					Namespace: "testns",
				},
				Status: integreatlyv1alpha1.RHMIStatus{
					ToVersion: "9.9.9",
					Version:   "8.8.8",
				},
			},
			Fields: fields{
				Client: utils.NewTestClient(scheme,
					&crov1alpha1.Postgres{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "testpg",
							Namespace: "testns",
						},
					},
					&crov1alpha1.Redis{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "testredis",
							Namespace: "testns",
						},
					},
				),
			},
			IsServiceAffecting:     true,
			ExpectedUpdatesAllowed: true,
		},
		{
			Name: "updates not allowed when Version is not empty, toVersion is empty and upgrade is service affecting",
			RHMI: integreatlyv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testrhmi",
					Namespace: "testns",
				},
				Status: integreatlyv1alpha1.RHMIStatus{
					Version: "8.8.8",
				},
			},
			Fields: fields{
				Client: utils.NewTestClient(scheme,
					&crov1alpha1.Postgres{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "testpg",
							Namespace: "testns",
						},
					},
					&crov1alpha1.Redis{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "testredis",
							Namespace: "testns",
						},
					},
				),
			},
			IsServiceAffecting:     true,
			ExpectedUpdatesAllowed: false,
		},
		{
			Name: "updates not allowed when Version is not empty, toVersion is not empty and upgrade is not service affecting",
			RHMI: integreatlyv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testrhmi",
					Namespace: "testns",
				},
				Status: integreatlyv1alpha1.RHMIStatus{
					ToVersion: "9.9.9",
					Version:   "8.8.8",
				},
			},
			Fields: fields{
				Client: utils.NewTestClient(scheme,
					&crov1alpha1.Postgres{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "testpg",
							Namespace: "testns",
						},
					},
					&crov1alpha1.Redis{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "testredis",
							Namespace: "testns",
						},
					},
				),
			},
			IsServiceAffecting:     false,
			ExpectedUpdatesAllowed: false,
		},
		{
			Name: "updates not allowed when Version is not empty, toVersion is empty and upgrade is not service affecting",
			RHMI: integreatlyv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testrhmi",
					Namespace: "testns",
				},
				Status: integreatlyv1alpha1.RHMIStatus{
					Version: "8.8.8",
				},
			},
			Fields: fields{
				Client: utils.NewTestClient(scheme,
					&crov1alpha1.Postgres{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "testpg",
							Namespace: "testns",
						},
					},
					&crov1alpha1.Redis{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "testredis",
							Namespace: "testns",
						},
					},
				),
			},
			IsServiceAffecting:     false,
			ExpectedUpdatesAllowed: false,
		},
		{
			Name: "updates not allowed when Version is empty, toVersion is not empty and upgrade is service affecting",
			RHMI: integreatlyv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testrhmi",
					Namespace: "testns",
				},
				Status: integreatlyv1alpha1.RHMIStatus{
					ToVersion: "9.9.9",
				},
			},
			Fields: fields{
				Client: utils.NewTestClient(scheme,
					&crov1alpha1.Postgres{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "testpg",
							Namespace: "testns",
						},
					},
					&crov1alpha1.Redis{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "testredis",
							Namespace: "testns",
						},
					},
				),
			},
			IsServiceAffecting:     true,
			ExpectedUpdatesAllowed: false,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			reconciler := &SubscriptionReconciler{
				Client:            scenario.Fields.Client,
				operatorNamespace: "testns",
			}
			err := reconciler.allowDatabaseUpdates(context.TODO(), &scenario.RHMI, scenario.IsServiceAffecting)
			if err != nil {
				t.Errorf("Unexpected error. Got %v", err)
			}

			isCorrect, err := allowUpdatesValueIsCorrect(scenario.Fields.Client, "testpg", "testredis", "testns", scenario.ExpectedUpdatesAllowed)
			if err != nil {
				t.Errorf("Unexpected error checking values in Postgres & Redis CRs, error: %v", err)
			}
			if !isCorrect {
				t.Errorf("Incorrect updatesAllowed value in Postgres or Redis CR")
			}
		})
	}
}

func allowUpdatesValueIsCorrect(client k8sclient.Client, postgresName, redisName, namespace string, want bool) (bool, error) {
	pg := crov1alpha1.Postgres{}
	if err := client.Get(context.TODO(), k8sclient.ObjectKey{
		Namespace: namespace,
		Name:      postgresName,
	}, &pg); err != nil {
		return false, err
	}

	redis := crov1alpha1.Redis{}
	if err := client.Get(context.TODO(), k8sclient.ObjectKey{
		Namespace: namespace,
		Name:      redisName,
	}, &redis); err != nil {
		return false, err
	}

	if pg.Spec.MaintenanceWindow != want || redis.Spec.MaintenanceWindow != want {
		return false, nil
	}
	return true, nil
}

func TestSubscriptionReconciler_HandleUpgrades(t *testing.T) {
	defaultCSV := &operatorsv1alpha1.ClusterServiceVersion{
		Spec: operatorsv1alpha1.ClusterServiceVersionSpec{
			Replaces: "1.2.3",
		},
	}
	defaultCSVStringfied, err := json.Marshal(defaultCSV)
	if err != nil {
		panic(err)
	}

	defaultInstallPlan := &operatorsv1alpha1.InstallPlan{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultInstallPlanName,
			Namespace: operatorNamespace,
		},
		Status: operatorsv1alpha1.InstallPlanStatus{
			Plan: []*operatorsv1alpha1.Step{
				{
					Resource: operatorsv1alpha1.StepResource{
						Kind:     operatorsv1alpha1.ClusterServiceVersionKind,
						Manifest: string(defaultCSVStringfied),
					},
				},
			},
		},
	}

	defaultInstallation := &integreatlyv1alpha1.RHMI{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rhmi",
			Namespace: operatorNamespace,
		},
	}

	defaultSubscription := &operatorsv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: operatorNamespace,
			Name:      IntegreatlyPackage,
		},
		Spec: &operatorsv1alpha1.SubscriptionSpec{
			InstallPlanApproval: operatorsv1alpha1.ApprovalAutomatic,
		},
		Status: operatorsv1alpha1.SubscriptionStatus{
			InstallPlanRef: &corev1.ObjectReference{
				Name:      defaultInstallPlan.Name,
				Namespace: defaultInstallPlan.Namespace,
			},
			CurrentCSV:   "1.2.4",
			InstalledCSV: "1.2.3",
		},
	}

	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatalf("failed to build scheme: %s", err.Error())
	}

	type fields struct {
		csv         *operatorsv1alpha1.ClusterServiceVersion
		installPlan *operatorsv1alpha1.InstallPlan
	}

	type args struct {
		rhmiSubscription *operatorsv1alpha1.Subscription
		installation     *integreatlyv1alpha1.RHMI
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    controllerruntime.Result
		wantErr bool
	}{
		{
			name: "HandleUpgrades should pass on valid input and already approved InstallPlan",
			args: args{
				rhmiSubscription: defaultSubscription,
				installation:     defaultInstallation,
			},
			fields: fields{
				csv: defaultCSV,
				installPlan: &operatorsv1alpha1.InstallPlan{
					ObjectMeta: metav1.ObjectMeta{
						Name:      defaultInstallPlanName,
						Namespace: operatorNamespace,
					},
					Spec: operatorsv1alpha1.InstallPlanSpec{
						Approved: true,
					},
					Status: operatorsv1alpha1.InstallPlanStatus{
						Plan: []*operatorsv1alpha1.Step{
							{
								Resource: operatorsv1alpha1.StepResource{
									Kind:     operatorsv1alpha1.ClusterServiceVersionKind,
									Manifest: string(defaultCSVStringfied),
								},
							},
						},
					},
				},
			},
			want: controllerruntime.Result{
				Requeue:      true,
				RequeueAfter: time.Minute,
			},
			wantErr: false,
		},
		{
			name: "HandleUpgrades should pass on valid input and unapproved InstallPlan",
			args: args{
				rhmiSubscription: defaultSubscription,
				installation:     defaultInstallation,
			},
			fields: fields{
				csv: defaultCSV,
				installPlan: &operatorsv1alpha1.InstallPlan{
					ObjectMeta: metav1.ObjectMeta{
						Name:      defaultInstallPlanName,
						Namespace: operatorNamespace,
					},
					Spec: operatorsv1alpha1.InstallPlanSpec{
						Approved: false,
					},
					Status: operatorsv1alpha1.InstallPlanStatus{
						Plan: []*operatorsv1alpha1.Step{
							{
								Resource: operatorsv1alpha1.StepResource{
									Kind:     operatorsv1alpha1.ClusterServiceVersionKind,
									Manifest: string(defaultCSVStringfied),
								},
							},
						},
					},
				},
			},
			want: controllerruntime.Result{
				Requeue:      true,
				RequeueAfter: time.Minute,
			},
			wantErr: false,
		},
		{
			name: "HandleUpgrades should fail on invalid CSV",
			args: args{
				rhmiSubscription: defaultSubscription,
				installation:     defaultInstallation,
			},

			fields: fields{
				csv: &operatorsv1alpha1.ClusterServiceVersion{},
				installPlan: &operatorsv1alpha1.InstallPlan{
					ObjectMeta: metav1.ObjectMeta{
						Name:      defaultInstallPlanName,
						Namespace: operatorNamespace,
					},
					Status: operatorsv1alpha1.InstallPlanStatus{
						Plan: []*operatorsv1alpha1.Step{
							{
								Resource: operatorsv1alpha1.StepResource{
									Kind: operatorsv1alpha1.ClusterServiceVersionKind,
								},
							},
						},
					},
				},
			},
			want:    controllerruntime.Result{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := utils.NewTestClient(scheme, tt.fields.csv, tt.fields.installPlan, tt.args.rhmiSubscription, tt.args.installation)
			r := &SubscriptionReconciler{
				Client:              client,
				Scheme:              scheme,
				operatorNamespace:   operatorNamespace,
				catalogSourceClient: getCatalogSourceClient(""),
				csvLocator:          &csvlocator.EmbeddedCSVLocator{},
			}
			got, err := r.HandleUpgrades(context.TODO(), tt.args.rhmiSubscription, tt.args.installation)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleUpgrades() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleUpgrades() got = %v, want %v", got, tt.want)
			}
		})
	}
}
