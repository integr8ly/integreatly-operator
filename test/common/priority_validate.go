package common

import (
	goctx "context"
	"fmt"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	openshiftappsv1 "github.com/openshift/api/apps/v1"
	appsv1 "k8s.io/api/apps/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func priorityStatefulSets(installType string) []StatefulSets {
	rhsso := StatefulSets{
		Namespace: NamespacePrefix + "rhsso",
		Name:      "keycloak",
	}
	usersso := StatefulSets{
		Namespace: NamespacePrefix + "user-sso",
		Name:      "keycloak",
	}

	if integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(installType)) {
		return []StatefulSets{rhsso}
	} else {
		return []StatefulSets{rhsso, usersso}
	}
}

func priorityDeploymentConfigs() []DeploymentConfigs {
	return []DeploymentConfigs{
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
			Name:      "system-sphinx",
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

// TestPriorityClass tests to ensure the pod priority class is created and verifies the deploymentconfigs and statefulsets are updated correctly
func TestPriorityClass(t TestingTB, ctx *TestingContext) {

	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("failed to get the RHMI: %s", err)
	}

	if !integreatlyv1alpha1.IsRHOAM(integreatlyv1alpha1.InstallationType(rhmi.Spec.Type)) {
		t.Skip("Skipping test as this is not a managed api install")
	}

	priorityClass := &schedulingv1.PriorityClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: rhmi.Spec.PriorityClassName,
		},
	}

	err = checkPriorityClassExists(priorityClass, ctx)
	if err != nil {
		t.Errorf("Error %v", err)
	}

	for _, priority := range priorityStatefulSets(rhmi.Spec.Type) {
		item, err := ctx.KubeClient.AppsV1().StatefulSets(priority.Namespace).Get(goctx.TODO(), priority.Name, metav1.GetOptions{})
		if err != nil {
			t.Errorf("Error: %v", err)
		}

		err = checkStatefulSetPriorityIsSet(item, rhmi.Spec.PriorityClassName)
		if err != nil {
			t.Errorf("Error %v", err)
		}
	}

	for _, priority := range priorityDeploymentConfigs() {
		deploymentConfig := &openshiftappsv1.DeploymentConfig{ObjectMeta: metav1.ObjectMeta{Name: priority.Name, Namespace: priority.Namespace}}
		err := ctx.Client.Get(goctx.TODO(), k8sclient.ObjectKey{Name: deploymentConfig.Name, Namespace: deploymentConfig.Namespace}, deploymentConfig)

		if err != nil {
			t.Errorf("Error: %v", err)
			break
		}

		err = checkDeploymentConfigPriorityIsSet(deploymentConfig, rhmi.Spec.PriorityClassName)
		if err != nil {
			t.Errorf("Error %v", err)
		}
	}
}

func checkStatefulSetPriorityIsSet(statefulSet *appsv1.StatefulSet, priorityClassName string) error {
	if statefulSet.Spec.Template.Spec.PriorityClassName != priorityClassName {
		return fmt.Errorf("priorityClassName is not set in statefulSet %v", statefulSet.Name)
	}
	return nil
}

func checkDeploymentConfigPriorityIsSet(deploymentConfig *openshiftappsv1.DeploymentConfig, priorityClassName string) error {
	if deploymentConfig.Spec.Template.Spec.PriorityClassName != priorityClassName {
		return fmt.Errorf("priorityClassName is not set in statefulSet %v", deploymentConfig.Name)
	}
	return nil
}

func checkPriorityClassExists(priorityClass *schedulingv1.PriorityClass, ctx *TestingContext) error {
	//err := ctx.Client.Get(goctx.TODO(), priorityClass, &k8sclient.ListOptions{Namespace: ""})

	_, err := ctx.KubeClient.SchedulingV1().PriorityClasses().Get(goctx.TODO(), priorityClass.Name, metav1.GetOptions{})

	if err != nil {
		return fmt.Errorf("priority Class : %v", err)

	}

	fmt.Print(priorityClass.Name)
	return nil
}
