package controllers

import (
	"context"
	"testing"

	cloudcredentialv1 "github.com/openshift/api/operator/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	operatorNamespace = "openshift-operators"
)

func getBuildScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	err := cloudcredentialv1.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}

	return scheme, err
}

func TestReconciler_checkIfStsClusterByCredentialsMode(t *testing.T) {
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatalf("Error obtaining scheme")
	}
	tests := []struct {
		name       string
		ARN        string
		fakeClient k8sclient.Client
		want       bool
		wantErr    bool
	}{
		{
			name: "STS cluster",
			fakeClient: fake.NewFakeClientWithScheme(scheme, &cloudcredentialv1.CloudCredential{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: cloudcredentialv1.CloudCredentialSpec{
					CredentialsMode: cloudcredentialv1.CloudCredentialsModeManual,
				},
			}),
			want:    true,
			wantErr: false,
		},
		{
			name: "Non STS cluster",
			fakeClient: fake.NewFakeClientWithScheme(scheme, &cloudcredentialv1.CloudCredential{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: cloudcredentialv1.CloudCredentialSpec{
					CredentialsMode: cloudcredentialv1.CloudCredentialsModeDefault,
				},
			}),
			want:    false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		got, err := checkIfStsClusterByCredentialsMode(context.TODO(), tt.fakeClient, operatorNamespace)
		if (err != nil) != tt.wantErr {
			t.Errorf("checkIfStsClusterByCredentialsMode() error = %v, wantErr %v", err, tt.wantErr)
			return
		}
		if got != tt.want {
			t.Errorf("checkIfStsClusterByCredentialsMode() got = %v, want %v", got, tt.want)
		}
	}
}
