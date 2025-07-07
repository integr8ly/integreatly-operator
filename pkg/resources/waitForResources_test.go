package resources

import (
	"fmt"

	"testing"

	crov1alpha1 "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1"
	crotypes "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1/types"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	"github.com/integr8ly/integreatly-operator/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestWaitForRHSSOPostgresToBeComplete(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	const installationName = "test"
	const installationNameSpace = "testNamespace"

	type args struct {
		serverClient     client.Client
		installName      string
		installNamespace string
	}
	tests := []struct {
		name    string
		args    args
		want    integreatlyv1alpha1.StatusPhase
		wantErr bool
	}{
		{
			name: "test phase complete when postgres phase complete",
			args: args{
				serverClient: utils.NewTestClient(scheme, &crov1alpha1.Postgres{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s%s", constants.RHSSOPostgresPrefix, installationName),
						Namespace: installationNameSpace,
					},
					Status: crotypes.ResourceTypeStatus{Phase: crotypes.PhaseComplete},
				}),
				installName:      installationName,
				installNamespace: installationNameSpace,
			},
			want: integreatlyv1alpha1.PhaseCompleted,
		},
		{
			name: "test phase awaiting components when postgres phase not complete",
			args: args{
				serverClient: utils.NewTestClient(scheme, &crov1alpha1.Postgres{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s%s", constants.RHSSOPostgresPrefix, installationName),
						Namespace: installationNameSpace,
					},
					Status: crotypes.ResourceTypeStatus{Phase: crotypes.PhaseInProgress},
				}),
				installName:      installationName,
				installNamespace: installationNameSpace,
			},
			want: integreatlyv1alpha1.PhaseAwaitingComponents,
		},
		{
			name: "test phase awaiting components when unable to get postgres cr",
			args: args{
				serverClient:     utils.NewTestClient(scheme),
				installName:      installationName,
				installNamespace: installationNameSpace,
			},
			want: integreatlyv1alpha1.PhaseAwaitingComponents,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := WaitForRHSSOPostgresToBeComplete(tt.args.serverClient, tt.args.installName, tt.args.installNamespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("WaitForRHSSOPostgresToBeComplete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("WaitForRHSSOPostgresToBeComplete() got = %v, want %v", got, tt.want)
			}
		})
	}
}
