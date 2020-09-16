package common

import (
	"testing"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	namespaces = []Namespace{
		{
			Name: NamespacePrefix + "rhsso",
			PodDisruptionBudgetNames: []string{
				"keycloak",
			},
		},
		{
			Name: NamespacePrefix + "user-sso",
			PodDisruptionBudgetNames: []string{
				"keycloak",
			},
		},
		// {
		// 	Name: NamespacePrefix + "3scale",
		// 	PodDisruptionBudgetNames: []string{
		// 		"apicast-production",
		// 		"apicast-staging",
		// 		"backend-cron",
		// 		"backend-listener",
		// 		"backend-worker",
		// 		"system-app",
		// 		"system-sidekiq",
		// 		"zync",
		// 		"zync-que",
		// 	},
		// },
	}
)

func TestIntegreatlyPodDisruptionBudgetsExist(t *testing.T, ctx *TestingContext) {
	for _, namespace := range namespaces {
		for _, podDisruptionBudgetName := range namespace.PodDisruptionBudgetNames {
			_, err := ctx.KubeClient.PolicyV1beta1().PodDisruptionBudgets(namespace.Name).Get(podDisruptionBudgetName, v1.GetOptions{})
			if err != nil {
				t.Errorf("PodDisruptionBudget %s not found in namespace %s - Error: %s", podDisruptionBudgetName, namespace.Name, err)
			}
		}
	}
}
