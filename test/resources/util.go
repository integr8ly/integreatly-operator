package resources

import (
	goctx "context"
	"fmt"
	k8errors "k8s.io/apimachinery/pkg/api/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"k8s.io/client-go/kubernetes"
)

func RunningInProw(inst *integreatlyv1alpha1.RHMI) bool {
	if v, ok := inst.Annotations["in_prow"]; !ok || v == "false" {
		return false
	}
	return true
}

func GetSMTPSecret(kubeClient kubernetes.Interface, operatorNamespace string, secretName string) (map[string][]byte, error) {
	res, err := kubeClient.CoreV1().Secrets(operatorNamespace).Get(goctx.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		if k8errors.IsNotFound(err) {
			return map[string][]byte{
				"username": []byte("smtp_username"),
				"password": []byte("smtp_password"),
				"host":     []byte("smtp.example.com"),
				"port":     []byte("587"),
			}, nil
		}
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}
	smtp := res.Data
	// sets default smtp host value as
	// it could not be set in the smtp secret to avoid issue with the 3scale api
	if string(smtp["username"]) == "" {
		smtp["username"] = []byte("smtp_username")
	}
	if string(smtp["password"]) == "" {
		smtp["password"] = []byte("smtp_password")
	}
	if string(smtp["host"]) == "" {
		smtp["host"] = []byte("smtp.example.com")
	}
	if string(smtp["port"]) == "" {
		smtp["port"] = []byte("587")
	}
	return smtp, nil
}
