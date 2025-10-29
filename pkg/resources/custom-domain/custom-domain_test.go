package custom_domain

import (
	"context"
	"errors"
	"net"
	"reflect"
	"testing"

	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/utils"
	customdomainv1alpha1 "github.com/openshift/custom-domains-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGetDomain(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	exampleNameSpace := "redhat-rhoam-operator"

	rhoamInstallation := &v1alpha1.RHMI{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "managed-api",
			Namespace: exampleNameSpace,
		},
		Spec: v1alpha1.RHMISpec{
			Type: "managed-api",
		},
	}

	type args struct {
		ctx          context.Context
		client       client.Client
		installation *v1alpha1.RHMI
	}
	tests := []struct {
		name    string
		args    args
		ok      bool
		domain  string
		wantErr bool
	}{
		{
			name: "Custom Domain successfully gotten",
			args: args{
				ctx: context.TODO(),
				client: utils.NewTestClient(scheme,
					&corev1.Secret{
						ObjectMeta: v1.ObjectMeta{
							Name:      "addon-managed-api-service-parameters",
							Namespace: exampleNameSpace,
						},
						Data: map[string][]byte{
							"custom-domain_domain": []byte("apps.example.com"),
						},
					},
				),
				installation: rhoamInstallation,
			},
			ok:      true,
			domain:  "apps.example.com",
			wantErr: false,
		},
		{
			name: "Custom Domain has leading/trailing white spaces",
			args: args{
				ctx: context.TODO(),
				client: utils.NewTestClient(scheme,
					&corev1.Secret{
						ObjectMeta: v1.ObjectMeta{
							Name:      "addon-managed-api-service-parameters",
							Namespace: exampleNameSpace,
						},
						Data: map[string][]byte{
							"custom-domain_domain": []byte("  apps.example.com  "),
						},
					}),
				installation: rhoamInstallation,
			},
			ok:      true,
			domain:  "apps.example.com",
			wantErr: false,
		},
		{
			name: "No Custom Domain set in addon secret",
			args: args{
				ctx: context.TODO(),
				client: utils.NewTestClient(scheme,
					&corev1.Secret{
						ObjectMeta: v1.ObjectMeta{
							Name:      "addon-managed-api-service-parameters",
							Namespace: exampleNameSpace,
						},
						Data: map[string][]byte{},
					}),
				installation: rhoamInstallation,
			},
			ok:      false,
			domain:  "",
			wantErr: false,
		},
		{
			name: "Invalid Custom Domain set in addon secret",
			args: args{
				ctx: context.TODO(),
				client: utils.NewTestClient(scheme,
					&corev1.Secret{
						ObjectMeta: v1.ObjectMeta{
							Name:      "addon-managed-api-service-parameters",
							Namespace: exampleNameSpace,
						},
						Data: map[string][]byte{
							"custom-domain_domain": []byte("bad domain"),
						},
					}),
				installation: rhoamInstallation,
			},
			ok:      true,
			domain:  "bad domain",
			wantErr: true,
		},
		{
			name: "Error getting addon secret",
			args: args{
				ctx: context.TODO(),
				client: utils.NewTestClient(scheme,
					&corev1.Secret{
						ObjectMeta: v1.ObjectMeta{
							Name:      "addon-dummy-service-parameters",
							Namespace: exampleNameSpace,
						},
						Data: map[string][]byte{},
					}),
				installation: rhoamInstallation,
			},
			ok:      false,
			domain:  "",
			wantErr: true,
		},
		{
			name:    "Nil pointer passed in for installation type",
			ok:      false,
			domain:  "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, domain, err := GetDomain(tt.args.ctx, tt.args.client, tt.args.installation)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDomain() error = %v, wantErr %v", err, tt.wantErr)
			}
			if ok != tt.ok {
				t.Errorf("GetDomain() ok = %v, wanted ok = %v", ok, tt.ok)
			}
			if domain != tt.domain {
				t.Errorf("GetDomain() domain = %v, wanted domain = %v", domain, tt.domain)
			}
		})
	}
}

func TestIsValidDomain(t *testing.T) {
	type args struct {
		domain string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Valid domain",
			args: args{domain: "good.domain.com"},
			want: true,
		},
		{
			name: "Invalid domain",
			args: args{domain: "bad domain.com"},
			want: false,
		},
		{
			name: "Domain with unwanted prefix",
			args: args{domain: "https://prefix.domain.com"},
			want: false,
		},
		{
			name: "Domain name with unwanted suffix",
			args: args{domain: "suffix.domain.com/"},
			want: false,
		},
		{
			name: "Blank domain passed to function",
			args: args{domain: ""},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidDomain(tt.args.domain); got != tt.want {
				t.Errorf("IsValidDomain() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasValidCustomDomainCR(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		ctx          context.Context
		serverClient client.Client
		domain       string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "Valid CustomDomain found (1 CR)",
			args: args{
				domain: "apps.example.com",
				ctx:    context.TODO(),
				serverClient: utils.NewTestClient(scheme, &customdomainv1alpha1.CustomDomainList{
					Items: []customdomainv1alpha1.CustomDomain{
						{
							ObjectMeta: v1.ObjectMeta{
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
			want:    true,
			wantErr: false,
		},
		{
			name: "Valid CustomDomain found (Multi CR)",
			args: args{
				domain: "apps.example.com",
				ctx:    context.TODO(),
				serverClient: utils.NewTestClient(scheme, &customdomainv1alpha1.CustomDomainList{
					Items: []customdomainv1alpha1.CustomDomain{
						{
							ObjectMeta: v1.ObjectMeta{
								Name: "badDomain",
							},
							Spec: customdomainv1alpha1.CustomDomainSpec{
								Domain: "bad.example.com",
							},
							Status: customdomainv1alpha1.CustomDomainStatus{
								State: "Failing",
							},
						},
						{
							ObjectMeta: v1.ObjectMeta{
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
			want:    true,
			wantErr: false,
		},
		{
			name:    "Empty/invalid domain string passed to function",
			args:    args{domain: ""},
			want:    false,
			wantErr: true,
		},
		{
			name: "CustomDomain CR not in Ready state",
			args: args{
				domain: "apps.example.com",
				ctx:    context.TODO(),
				serverClient: utils.NewTestClient(scheme, &customdomainv1alpha1.CustomDomainList{
					Items: []customdomainv1alpha1.CustomDomain{
						{
							ObjectMeta: v1.ObjectMeta{
								Name: "goodDomain",
							},
							Spec: customdomainv1alpha1.CustomDomainSpec{
								Domain: "apps.example.com",
							},
							Status: customdomainv1alpha1.CustomDomainStatus{
								State: "Failing",
							},
						},
					},
				}),
			},
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := HasValidCustomDomainCR(tt.args.ctx, tt.args.serverClient, tt.args.domain)
			if (err != nil) != tt.wantErr {
				t.Errorf("HasValidCustomDomainCR() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("HasValidCustomDomainCR() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpdateErrorAndMetric(t *testing.T) {
	rhoamInstallation := &v1alpha1.RHMI{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "managed-api",
			Namespace: "redhat-rhoam-operator",
		},
		Status: v1alpha1.RHMIStatus{
			CustomDomain: &v1alpha1.CustomDomainStatus{},
		},
	}
	type args struct {
		installation *v1alpha1.RHMI
		active       bool
		err          error
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "update metric and clear custom domain error",
			args: args{
				installation: rhoamInstallation,
				active:       true,
				err:          nil,
			},
		},
		{
			name: "update metric and set custom domain error",
			args: args{
				installation: rhoamInstallation,
				active:       false,
				err:          errors.New("generic error"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			UpdateErrorAndCustomDomainMetric(tt.args.installation, tt.args.active, tt.args.err)
		})
	}
}

func TestGetIngressRouterService(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		ctx          context.Context
		serverClient func() client.Client
	}
	tests := []struct {
		name    string
		args    args
		want    *corev1.Service
		wantErr bool
	}{
		{
			name: "successfully retrieve ingress router service",
			args: args{
				ctx: context.TODO(),
				serverClient: func() client.Client {
					return moqclient.NewSigsClientMoqWithScheme(scheme,
						&corev1.Service{
							ObjectMeta: v1.ObjectMeta{
								Name:      "router-default",
								Namespace: "openshift-ingress",
							},
						})
				},
			},
			want:    &corev1.Service{},
			wantErr: false,
		},
		{
			name: "failed to retrieve ingress router service",
			args: args{
				ctx: context.TODO(),
				serverClient: func() client.Client {
					mockClient := moqclient.NewSigsClientMoqWithScheme(scheme)
					mockClient.GetFunc = func(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) error {
						return errors.New("generic error")
					}
					return mockClient
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetIngressRouterService(tt.args.ctx, tt.args.serverClient(), "router-default")
			if (err != nil) != tt.wantErr {
				t.Errorf("GetIngressRouterService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (got != nil) != (tt.want != nil) {
				t.Errorf("GetIngressRouterService() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetIngressRouterIPs(t *testing.T) {
	type args struct {
		loadBalancerIngress []corev1.LoadBalancerIngress
	}
	tests := []struct {
		name    string
		args    args
		want    []net.IP
		wantErr bool
	}{
		{
			name: "failed to perform ip lookup for hostname",
			args: args{
				loadBalancerIngress: []corev1.LoadBalancerIngress{
					{
						Hostname: "hostname",
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "valid ip address in load balancer",
			args: args{
				loadBalancerIngress: []corev1.LoadBalancerIngress{
					{
						IP: "0.0.0.0",
					},
				},
			},
			want:    []net.IP{net.ParseIP("0.0.0.0")},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetIngressRouterIPs(tt.args.loadBalancerIngress)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetIngressRouterIPs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetIngressRouterIPs() got = %v, want %v", got, tt.want)
			}
		})
	}
}
