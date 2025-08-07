package common

import (
	"context"
	"fmt"
	"github.com/integr8ly/integreatly-operator/api/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

var (
	namespaceToVerify = []string{
		RHOAMOperatorNamespace,
		CloudResourceOperatorNamespace,
		ObservabilityProductNamespace,
		CustomerGrafanaNamespace,
		RHSSOUserProductNamespace,
		RHSSOUserOperatorNamespace,
		RHSSOProductNamespace,
		RHSSOOperatorNamespace,
		Marin3rOperatorNamespace,
		Marin3rProductNamespace,
		ThreeScaleProductNamespace,
		ThreeScaleOperatorNamespace,
	}
	productsToTest = map[v1alpha1.ProductName]map[string]string{
		v1alpha1.Product3Scale: {
			quota.BackendListenerName:   "deployment=backend-listener",
			quota.BackendWorkerName:     "deployment=backend-worker",
			quota.ApicastProductionName: "deployment=apicast-production",
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
	// Verify that all pods in every RHOAM namespace have requests set for both CPU and memory
	verifyPodRequests(t, ctx)

	// Verify that the RHOAM has correctly matching the quota config
	verifyQuotaConfig(t, ctx)
}

func verifyPodRequests(t TestingTB, ctx *TestingContext) {
	for _, ns := range namespaceToVerify {
		pods := &corev1.PodList{}
		listOpts := []k8sclient.ListOption{
			k8sclient.InNamespace(ns),
		}

		err := ctx.Client.List(context.TODO(), pods, listOpts...)
		if err != nil {
			t.Fatal(fmt.Sprintf("Failed to get list of Pods from Namespace %s: %s", ns, err))
		}

		for _, pod := range pods.Items {
			for _, initContainer := range pod.Spec.InitContainers {
				if !areRequestsSet(initContainer.Resources.Requests) {
					t.Fatal(fmt.Sprintf("InitContainer %s from Pod %s in Namespace %s does not have CPU and/or memory requests set", initContainer.Name, pod.Name, ns))
				}
			}
			for _, container := range pod.Spec.Containers {
				if !areRequestsSet(container.Resources.Requests) {
					t.Fatal(fmt.Sprintf("Container %s from Pod %s in Namespace %s does not have CPU and/or memory requests set", container.Name, pod.Name, ns))
				}
			}

		}
	}
}

func areRequestsSet(requests corev1.ResourceList) bool {
	return requests != nil && requests.Memory() != nil && requests.Cpu() != nil
}

func verifyQuotaConfig(t TestingTB, ctx *TestingContext) {
	quotaConfig, err := getQuotaConfig(t, ctx.Client)
	if err != nil {
		t.Fatal("Error getting quota")
		return
	}

	for product, pods := range productsToTest {
		productConfig := quotaConfig.GetProduct(product)

		for pod, label := range pods {
			var resourceConfig corev1.ResourceRequirements
			var ok bool

			// in order to simplify logic and data structure one of pods represent a product namespace
			if pod != "namespace" {
				resourceConfig, ok = productConfig.GetResourceConfig(pod)
				if !ok {
					t.Fatal(fmt.Sprintf("Couldn't retrieve config for %s pod from %s product config", pod, product))
					return
				}

				clientPods := &corev1.PodList{}
				selector, err := labels.Parse(label)
				if err != nil {
					t.Fatal(err)
				}

				listOpts := []k8sclient.ListOption{
					k8sclient.InNamespace(pods["namespace"]),
					k8sclient.MatchingLabelsSelector{
						Selector: selector,
					},
				}

				err = ctx.Client.List(context.TODO(), clientPods, listOpts...)
				if err != nil {
					t.Fatal(fmt.Sprintf("failed to get %s pods: %s", pod, err))
				}

				if !ensurePodMatchesConfig(t, clientPods, &resourceConfig) {
					t.Fatal(fmt.Sprintf("Found error in resource requirement for %s pod or it's containers", clientPods.Items[0].Name))
				}
			}

		}
	}
}

func ensurePodMatchesConfig(t TestingTB, pods *corev1.PodList, resourceRequirements *corev1.ResourceRequirements) bool {
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
		}
	}
	return true
}
