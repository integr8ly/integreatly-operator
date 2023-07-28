package obo

import (
	"context"
	"fmt"
	"github.com/integr8ly/integreatly-operator/pkg/resources/k8s"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	packageoperatorv1alpha1 "package-operator.run/apis/core/v1alpha1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	RHOAMOboClusterPackageName         = "managed-api-service"
	RHOAMInternalOboClusterPackageName = "managed-api-service-internal"
)

func GetOboClusterPackage(client k8sclient.Client) (*packageoperatorv1alpha1.ClusterPackage, error) {
	clusterPackageName, err := getClusterPackageName()
	if err != nil {
		return nil, err
	}

	clusterPackage := &packageoperatorv1alpha1.ClusterPackage{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterPackageName,
		},
	}
	err = client.Get(context.TODO(), k8sclient.ObjectKey{Name: clusterPackage.Name}, clusterPackage)
	if err != nil {
		return nil, err
	}

	return clusterPackage, nil
}

func GetOboClusterPackageLabel(client k8sclient.Client) (string, error) {
	clusterPackageName, err := getClusterPackageName()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("package-operator.run/package=%s", clusterPackageName), nil
}

func getClusterPackageName() (string, error) {
	watchNS, err := k8s.GetWatchNamespace()
	if err != nil {
		return "", err
	}

	clusterPackageName := RHOAMOboClusterPackageName
	if watchNS == "redhat-rhoami-operator" {
		clusterPackageName = RHOAMInternalOboClusterPackageName
	}

	return clusterPackageName, nil
}
