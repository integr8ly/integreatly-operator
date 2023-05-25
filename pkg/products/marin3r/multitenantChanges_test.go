package marin3r

import (
	"context"
	"errors"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/utils"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"testing"
)

func Test_csvUpdater_findCsv(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		ctx       context.Context
		client    client.Client
		namespace string
		log       logger.Logger
		csv       *operatorsv1alpha1.ClusterServiceVersion
	}
	tests := []struct {
		name        string
		fields      fields
		wantErr     bool
		expectedErr string
	}{
		{
			name:        "failed to list csv",
			wantErr:     true,
			expectedErr: "list function failure",
			fields: fields{
				log:       getLogger(),
				namespace: "local-test",
				ctx:       context.TODO(),
				client: func() client.Client {
					mockClient := moqclient.NewSigsClientMoqWithScheme(scheme)
					mockClient.ListFunc = func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
						return errors.New("list function failure")
					}
					return mockClient
				}(),
			},
		},
		{
			name:        "failed to find csv",
			wantErr:     true,
			expectedErr: "failed to find marin3r CSV",
			fields: fields{
				log:       getLogger(),
				namespace: "local-test",
				ctx:       context.TODO(),
				client: utils.NewTestClient(scheme,
					&operatorsv1alpha1.ClusterServiceVersionList{
						Items: []operatorsv1alpha1.ClusterServiceVersion{
							{
								ObjectMeta: metav1.ObjectMeta{Name: "some other csv", Namespace: "local-test"},
							},
						},
					}),
			},
		},
		{
			name:    "successfully find csv",
			wantErr: false,
			fields: fields{
				log:       getLogger(),
				namespace: "local-test",
				ctx:       context.TODO(),
				client: utils.NewTestClient(scheme,
					&operatorsv1alpha1.ClusterServiceVersionList{
						Items: []operatorsv1alpha1.ClusterServiceVersion{
							{
								ObjectMeta: metav1.ObjectMeta{Name: "some other csv", Namespace: "local-test"},
							},
							{
								ObjectMeta: metav1.ObjectMeta{Name: "marin3r-sample-csv", Namespace: "local-test"},
							},
						},
					}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &csvUpdater{
				ctx:       tt.fields.ctx,
				client:    tt.fields.client,
				namespace: tt.fields.namespace,
				log:       tt.fields.log,
				csv:       tt.fields.csv,
			}

			err := f.findCsv()
			if err != nil {
				if !tt.wantErr {
					t.Errorf("findCsv() error = %v, wantErr %v", err, tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.expectedErr) {
					t.Fatalf("findCsv() error = %v, should contain: %v", err, tt.expectedErr)
				}
			}
		})
	}
}

func Test_csvUpdater_setManagerResources(t *testing.T) {
	type fields struct {
		ctx       context.Context
		client    client.Client
		namespace string
		log       logger.Logger
		csv       *operatorsv1alpha1.ClusterServiceVersion
	}
	tests := []struct {
		name        string
		fields      fields
		wantErr     bool
		expectedErr string
	}{
		{
			name:        "function ran before csv is set",
			wantErr:     true,
			expectedErr: "csvUpdater.csv is not set",
			fields: fields{
				log:       getLogger(),
				namespace: "local-test",
			},
		},
		{
			name:        "manager container not updated",
			wantErr:     true,
			expectedErr: "unable to find manager container",
			fields: fields{
				log:       getLogger(),
				namespace: "local-test",
				csv: &operatorsv1alpha1.ClusterServiceVersion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-csv",
						Namespace: "test-local",
					},
				},
			},
		},
		{
			name:    "csv updated successfully",
			wantErr: false,
			fields: fields{
				log:       getLogger(),
				namespace: "local-test",
				csv: &operatorsv1alpha1.ClusterServiceVersion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-csv",
						Namespace: "test-local",
					},
					Spec: operatorsv1alpha1.ClusterServiceVersionSpec{
						InstallStrategy: operatorsv1alpha1.NamedInstallStrategy{
							StrategySpec: operatorsv1alpha1.StrategyDetailsDeployment{
								DeploymentSpecs: []operatorsv1alpha1.StrategyDeploymentSpec{
									{
										Name: "marin3r-controller-manager",
										Spec: appsv1.DeploymentSpec{
											Template: corev1.PodTemplateSpec{
												Spec: corev1.PodSpec{
													Containers: []corev1.Container{
														{Name: "manager"},
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
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &csvUpdater{
				ctx:       tt.fields.ctx,
				client:    tt.fields.client,
				namespace: tt.fields.namespace,
				log:       tt.fields.log,
				csv:       tt.fields.csv,
			}

			err := f.setManagerResources()
			if err != nil {
				if !tt.wantErr {
					t.Errorf("setManagerResources() error = %v, wantErr %v", err, tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.expectedErr) {
					t.Fatalf("setManagerResources() error = %v, should contain: %v", err, tt.expectedErr)
				}
			}
		})
	}
}

func Test_csvUpdater_updateCSV(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		ctx       context.Context
		client    client.Client
		namespace string
		log       logger.Logger
		csv       *operatorsv1alpha1.ClusterServiceVersion
	}
	tests := []struct {
		name        string
		fields      fields
		wantErr     bool
		expectedErr string
		fetchCsv    bool
	}{
		{
			name:        "function ran before csv is set",
			wantErr:     true,
			expectedErr: "csvUpdater.csv is not set",
			fields: fields{
				log:    getLogger(),
				ctx:    context.TODO(),
				client: utils.NewTestClient(scheme),
			},
		},
		{
			name:        "fail to update csv",
			wantErr:     true,
			expectedErr: "\"test-csv\" not found",
			fields: fields{
				log:    getLogger(),
				ctx:    context.TODO(),
				client: utils.NewTestClient(scheme),
				csv: &operatorsv1alpha1.ClusterServiceVersion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-csv",
						Namespace: "test-local",
					},
				},
			},
		},
		{
			name:     "successfully updated csv",
			wantErr:  false,
			fetchCsv: true,
			fields: fields{
				log: getLogger(),
				ctx: context.TODO(),
				client: utils.NewTestClient(scheme,
					&operatorsv1alpha1.ClusterServiceVersion{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-csv",
							Namespace: "test-local",
						},
					}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &csvUpdater{
				ctx:       tt.fields.ctx,
				client:    tt.fields.client,
				namespace: tt.fields.namespace,
				log:       tt.fields.log,
				csv:       tt.fields.csv,
			}

			if tt.fetchCsv {
				csv := &operatorsv1alpha1.ClusterServiceVersion{}
				err = f.client.Get(f.ctx, client.ObjectKey{
					Name:      "test-csv",
					Namespace: "test-local",
				}, csv)

				if err != nil {
					t.Errorf("\"updateCSV() unexpected error = %v", err)
				}
				f.csv = csv
			}
			err = f.updateCSV()
			if err != nil {
				if !tt.wantErr {
					t.Errorf("updateCSV() error = %v, wantErr %v", err, tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.expectedErr) {
					t.Fatalf("updateCSV() error = %v, should contain: %v", err, tt.expectedErr)
				}
			}
		})
	}
}

func Test_newCsvUpdater(t *testing.T) {
	type args struct {
		ctx       context.Context
		client    client.Client
		namespace string
		log       logger.Logger
	}
	tests := []struct {
		name string
		args args
		want csvUpdater
	}{
		{
			name: "happy path",
			args: args{
				ctx:       context.TODO(),
				client:    nil,
				namespace: "test-namespace",
				log:       getLogger(),
			},
			want: csvUpdater{
				ctx:       context.TODO(),
				client:    nil,
				namespace: "test-namespace",
				log:       getLogger(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newCsvUpdater(tt.args.ctx, tt.args.client, tt.args.namespace, tt.args.log); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newCsvUpdater() = %v, want %v", got, tt.want)
			}
		})
	}
}
