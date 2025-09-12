package common

import (
	"context"
	"fmt"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

func priorityStatefulSets(installType string) []CustomResource {
	rhsso := CustomResource{
		Namespace: NamespacePrefix + "rhsso",
		Name:      "keycloak",
	}
	if integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(installType)) {
		return []CustomResource{rhsso}
	} else {
		usersso := CustomResource{
			Namespace: NamespacePrefix + "user-sso",
			Name:      "keycloak",
		}
		return []CustomResource{rhsso, usersso}
	}
}

func priorityDeployments() []CustomResource {
	return []CustomResource{
		{
			Namespace: NamespacePrefix + "3scale",
			Name:      "marin3r-instance",
		},
		{
			Namespace: NamespacePrefix + "3scale-operator",
			Name:      "threescale-operator-controller-manager-v2",
		},
		{
			Namespace: NamespacePrefix + "cloud-resources-operator",
			Name:      "cloud-resource-operator",
		},
		{
			Namespace: NamespacePrefix + "marin3r",
			Name:      "ratelimit",
		},
		{
			Namespace: NamespacePrefix + "marin3r-operator",
			Name:      "marin3r-controller-webhook",
		},
		{
			Namespace: NamespacePrefix + "marin3r-operator",
			Name:      "marin3r-controller-manager",
		},
		{
			Namespace: NamespacePrefix + "rhsso-operator",
			Name:      "rhsso-operator",
		},
		{
			Namespace: NamespacePrefix + "user-sso-operator",
			Name:      "rhsso-operator",
		},
		{
			Namespace: NamespacePrefix + "3scale",
			Name:      "apicast-production",
		},
		{
			Namespace: NamespacePrefix + "3scale",
			Name:      "apicast-staging",
		},
		{
			Namespace: NamespacePrefix + "3scale",
			Name:      "backend-cron",
		},
		{
			Namespace: NamespacePrefix + "3scale",
			Name:      "backend-listener",
		},
		{
			Namespace: NamespacePrefix + "3scale",
			Name:      "backend-worker",
		},
		{
			Namespace: NamespacePrefix + "3scale",
			Name:      "system-app",
		},
		{
			Namespace: NamespacePrefix + "3scale",
			Name:      "system-memcache",
		},
		{
			Namespace: NamespacePrefix + "3scale",
			Name:      "system-sidekiq",
		},
		{
			Namespace: NamespacePrefix + "3scale",
			Name:      "system-searchd",
		},
		{
			Namespace: NamespacePrefix + "3scale",
			Name:      "zync",
		},
		{
			Namespace: NamespacePrefix + "3scale",
			Name:      "zync-database",
		},
		{
			Namespace: NamespacePrefix + "3scale",
			Name:      "zync-que",
		},
	}
}

// TestPriorityClass tests to ensure the pod priority class is created and verifies various crs are updated accordingly
func TestPriorityClass(t TestingTB, ctx *TestingContext) {
	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("failed to get the RHMI: %s", err)
	}
	priorityClass := rhmi.Spec.PriorityClassName
	if err = checkPriorityClassExists(ctx, priorityClass); err != nil {
		t.Errorf(err.Error())
	}
	for _, ss := range priorityStatefulSets(rhmi.Spec.Type) {
		statefulSet := &appsv1.StatefulSet{}
		if err = ctx.Client.Get(context.TODO(), k8sclient.ObjectKey{Name: ss.Name, Namespace: ss.Namespace}, statefulSet); err != nil {
			t.Errorf("Error: %s", err.Error())
			break
		}
		if err = checkPriorityIsSet(statefulSet.Spec.Template.Spec, priorityClass); err != nil {
			t.Errorf("failure validating %s/%s: %s", statefulSet.Kind, statefulSet.Name, err.Error())
		}
	}
	for _, d := range priorityDeployments() {
		// skip the user-sso-operator deployment if multi-tenant installation
		if d.Name == "rhsso-operator" && strings.HasSuffix(d.Namespace, "user-sso-operator") {
			if integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(rhmi.Spec.Type)) {
				continue
			}
		}
		deployment := &appsv1.Deployment{}
		if err = ctx.Client.Get(context.TODO(), k8sclient.ObjectKey{Name: d.Name, Namespace: d.Namespace}, deployment); err != nil {
			t.Errorf("Error: %v", err)
			break
		}
		if err = checkPriorityIsSet(deployment.Spec.Template.Spec, priorityClass); err != nil {
			t.Errorf("failure validating %s/%s: %s", deployment.Kind, deployment.Name, err.Error())
		}
	}
}

func checkPriorityIsSet(spec corev1.PodSpec, priorityClassName string) error {
	if spec.PriorityClassName != priorityClassName {
		return fmt.Errorf("priorityClassName is not set")
	}
	return nil
}

func checkPriorityClassExists(ctx *TestingContext, priorityClassName string) error {
	priorityClass := &schedulingv1.PriorityClass{}
	if err := ctx.Client.Get(context.TODO(), k8sclient.ObjectKey{Name: priorityClassName}, priorityClass); err != nil {
		return fmt.Errorf("failure fetching priority class: %w", err)
	}
	return nil
}
