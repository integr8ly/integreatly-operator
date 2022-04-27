package common

import (
	"context"
	"fmt"
	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

var (
	productsToTest = map[v1alpha1.ProductName]map[string]string{
		v1alpha1.Product3Scale: {
			quota.BackendListenerName:   "deploymentconfig=backend-listener",
			quota.BackendWorkerName:     "deploymentconfig=backend-worker",
			quota.ApicastProductionName: "deploymentconfig=apicast-production",
			"namespace":                 ThreeScaleProductNamespace,
		},
		v1alpha1.ProductRHSSOUser: {
			quota.KeycloakName: "component=keycloak",
			"namespace":        RHSSOUserProductNamespace,
		},
		v1alpha1.ProductMarin3r: {
			quota.RateLimitName: "app=ratelimit",
			"namespace":         Marin3rProductNamespace,
		},
	}
)

func ValidateResourceRequirements(t TestingTB, ctx *TestingContext) {

	// We want to ensure that limits and requests are set, but it won't hurt to ensure that those limits match active quota
	quotaConfig, err := getQuotaConfig(t, ctx.Client)
	if err != nil {
		t.Fatal("Error getting quota")
		return
	}

	for product, pods := range productsToTest {
		productConfig := quotaConfig.GetProduct(product)

		for pod, label := range pods {
			var resourceConfig v1.ResourceRequirements
			var ok bool

			// in order to simplify logic and data structure one of pods represent a product namespace
			if pod != "namespace" {
				resourceConfig, ok = productConfig.GetResourceConfig(pod)
				if !ok {
					t.Fatal(fmt.Sprintf("Couldn't retrieve config for %s pod from %s product config", pod, product))
					return
				}

				clientPods := &v1.PodList{}
				selector, _ := labels.Parse(label)
				listOpts := []k8sclient.ListOption{
					k8sclient.InNamespace(pods["namespace"]),
					k8sclient.MatchingLabelsSelector{
						Selector: selector,
					},
				}

				err = ctx.Client.List(context.TODO(), clientPods, listOpts...)
				if err != nil {
					t.Fatalf(fmt.Sprintf("failed to get %s pods: %s", pod, err))
				}

				if !ensurePodMatchesConfig(clientPods, &resourceConfig, t) {
					t.Fatal(fmt.Sprintf("Found error in resource requirement for %s pod or it's containers", clientPods.Items[0].Name))
				}
			}

		}
	}
}

// there is existing function podMatchesConfig but it does not checks all the containers
// for this test we also want to be sure that sidecars have limits set
func ensurePodMatchesConfig(pods *v1.PodList, resourceRequirements *v1.ResourceRequirements, t TestingTB) bool {
	// all pods will share requests/limits so no point in checking all
	for _, container := range pods.Items[0].Spec.Containers {
		if strings.Contains(pods.Items[0].Name, container.Name) {
			if container.Resources.Limits.Cpu().Cmp(*resourceRequirements.Limits.Cpu()) != 0 &&
				container.Resources.Limits.Memory().Cmp(*resourceRequirements.Limits.Memory()) != 0 &&
				container.Resources.Requests.Memory().Cmp(*resourceRequirements.Requests.Memory()) != 0 &&
				container.Resources.Requests.Cpu().Cmp(*resourceRequirements.Requests.Cpu()) != 0 {
				t.Error(fmt.Sprintf("mismatch in container %s from %s pod. Does not match quota config", container.Name, pods.Items[0].Name))
				return false
			}
		} else {
			if container.Resources.Limits == nil || container.Resources.Requests == nil {
				t.Error(fmt.Sprintf("Sidecar container for pod %s does not have limits or requests set", pods.Items[0].Name))
				return false
			}
		}
	}
	return true
}
