package common

import (
	"testing"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type podDisruptionBudgetTestExecutor struct {
	namespaces []Namespace
}

var PodDisruptionBudgetTest TestCase = podDisruptionBudgetTestExecutor{
	namespaces: []Namespace{
		{
			Name: "redhat-rhmi-rhsso",
			PodDisruptionBudgetNames: []string{
				"keycloak",
			},
		},
		{
			Name: "redhat-rhmi-user-sso",
			PodDisruptionBudgetNames: []string{
				"keycloak",
			},
		},
		// {
		// 	Name: "redhat-rhmi-3scale",
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
	},
}

func (tc podDisruptionBudgetTestExecutor) Description() string {
	return "Verify PodDisruptionBudgets exist"
}

func (tc podDisruptionBudgetTestExecutor) Test(t *testing.T, ctx *TestingContext) {
	for _, namespace := range tc.namespaces {
		for _, podDisruptionBudgetName := range namespace.PodDisruptionBudgetNames {
			_, err := ctx.KubeClient.PolicyV1beta1().PodDisruptionBudgets(namespace.Name).Get(podDisruptionBudgetName, v1.GetOptions{})
			if err != nil {
				t.Errorf("PodDisruptionBudget %s not found in namespace %s - Error: %s", podDisruptionBudgetName, namespace.Name, err)
			}
		}
	}
}
