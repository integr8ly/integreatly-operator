package common

import (
	goctx "context"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"testing"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	clonedServiceMonitorLabelKey   = "integreatly.org/cloned-servicemonitor"
	clonedServiceMonitorLabelValue = "true"
	labelSelector                  = "monitoring-key=middleware"
	roleBindingName                = "rhmi-prometheus-k8s"
	roleRefName                    = "rhmi-prometheus-k8s"
)

// Specific to rmhi install type
var rhmi2ExpectedServiceMonitors = []string{
	"redhat-rhmi-amq-online-enmasse-address-space-controller",
	"redhat-rhmi-amq-online-enmasse-admin",
	"redhat-rhmi-amq-online-enmasse-broker",
	"redhat-rhmi-amq-online-enmasse-console",
	"redhat-rhmi-amq-online-enmasse-iot",
	"redhat-rhmi-amq-online-enmasse-operator-metrics",
	"redhat-rhmi-amq-online-enmasse-router",
	"redhat-rhmi-fuse-syndesis-infra",
	"redhat-rhmi-fuse-syndesis-integrations",
	"redhat-rhmi-ups-operator-unifiedpush-operator-metrics",
	"redhat-rhmi-ups-unifiedpush",
}

// Common to all install types
var commonExpectedServiceMonitors = []string{
	"redhat-rhmi-cloud-resources-operator-cloud-resource-operator-metrics",
	"redhat-rhmi-middleware-monitoring-operator-application-monitoring-operator-metrics",
	"redhat-rhmi-middleware-monitoring-operator-grafana-servicemonitor",
	"redhat-rhmi-middleware-monitoring-operator-prometheus-servicemonitor",
	"redhat-rhmi-rhsso-keycloak-service-monitor",
	"redhat-rhmi-user-sso-keycloak-service-monitor",
}

// TestServiceMonitorsCloneAndRolebindingsExist monitoring spec testcase
// Verifies the list of servicemonitors that are cloned in monitoring namespace
// Verifies the rolebindings exist
// Verifies if there are any stale service monitors in the monitoring namespace
func TestServiceMonitorsCloneAndRolebindingsExist(t *testing.T, ctx *TestingContext) {
	rhmi, err := getRHMI(ctx.Client)
	if err != nil {
		t.Fatalf("failed to get the RHMI: %s", err)
	}
	expectedServiceMonitors := getExpectedServiceMonitors(rhmi.Spec.Type)

	//Get list of service monitors in the monitoring namespace
	monSermonMap, err := getServiceMonitors(ctx, MonitoringSpecNamespace)
	if err != nil {
		t.Fatal("failed to list servicemonitors in monitoring namespace:", err)
	}
	if len(monSermonMap) == 0 {
		t.Fatal("No servicemonitors present in monitoring namespace")
	}
	//Validate the servicemonitors against the list
	for _, sm := range expectedServiceMonitors {
		if _, ok := monSermonMap[sm]; !ok {
			t.Fatal("Error - Servicemonitor(s) not found in monitoring namespace", sm)
		}
	}
	//Check if rolebindings exists
	for _, sm := range monSermonMap {
		for _, namespace := range sm.Spec.NamespaceSelector.MatchNames {
			err := checkRoleExists(ctx, roleRefName, namespace)
			if err != nil {
				t.Fatal("Error retrieving role: ", err, "in namespace:", namespace)
			}
			err = checkRoleBindingExists(ctx, roleBindingName, namespace)
			if err != nil {
				t.Fatal("Error retrieving rolebinding: ", err, "in namespace:", namespace)
			}
		}
	}
	//Get the namespaces
	ls, err := labels.Parse(labelSelector)
	if err != nil {
		t.Fatal("failed to parse label", err)
	}
	opts := &k8sclient.ListOptions{
		LabelSelector: ls,
	}
	namespaces := &corev1.NamespaceList{}
	err = ctx.Client.List(goctx.TODO(), namespaces, opts)
	if err != nil {
		t.Fatal("failed to list namespaces", err)
	}
	//Get servicemonitors for each namespace and validate them
	for _, ns := range namespaces.Items {
		//Get list of service monitors in each name space
		listOpts := []k8sclient.ListOption{
			k8sclient.InNamespace(ns.Name),
		}
		serviceMonitors := &monitoringv1.ServiceMonitorList{}
		err := ctx.Client.List(goctx.TODO(), serviceMonitors, listOpts...)
		if err != nil {
			t.Fatal("failed to list servicemonitors", err)
		}
		for _, sm := range serviceMonitors.Items {
			key := sm.Namespace + `-` + sm.Name
			if _, ok := monSermonMap[key]; !ok {
				t.Fatal("Servicemonitor: ", key, "not found in monitoring namespace")
			}
			delete(monSermonMap, key) // Servicemonitor exists, remove it from the local map
		}
	}
	// Any values left in the servicemonitors map are stale
	if len(monSermonMap) > 0 {
		var staleMonitors string
		for key := range monSermonMap {
			staleMonitors = staleMonitors + key + ","
		}
		t.Fatal("stale service monitors present: ", staleMonitors)
	}
}

func getServiceMonitors(ctx *TestingContext,
	nameSpace string) (serviceMonitorsMap map[string]*monitoringv1.ServiceMonitor, err error) {
	//Get list of service monitors in the namespace that has
	//label "integreatly.org/cloned-servicemonitor" set to "true"
	listOpts := []k8sclient.ListOption{
		k8sclient.InNamespace(nameSpace),
		k8sclient.MatchingLabels(getClonedServiceMonitorLabel()),
	}
	serviceMonitors := &monitoringv1.ServiceMonitorList{}
	err = ctx.Client.List(goctx.TODO(), serviceMonitors, listOpts...)
	if err != nil {
		return serviceMonitorsMap, err
	}
	serviceMonitorsMap = make(map[string]*monitoringv1.ServiceMonitor)
	for _, sm := range serviceMonitors.Items {
		serviceMonitorsMap[sm.Name] = sm
	}
	return serviceMonitorsMap, err
}

func getClonedServiceMonitorLabel() map[string]string {
	return map[string]string{
		clonedServiceMonitorLabelKey: clonedServiceMonitorLabelValue,
	}
}

func checkRoleBindingExists(ctx *TestingContext, name, namespace string) (err error) {
	rb := &rbac.RoleBinding{}
	err = ctx.Client.Get(goctx.TODO(), k8sclient.ObjectKey{Name: name, Namespace: namespace}, rb)
	return err
}

func checkRoleExists(ctx *TestingContext, name, namespace string) (err error) {
	role := &rbac.Role{}
	err = ctx.Client.Get(goctx.TODO(), k8sclient.ObjectKey{Name: name, Namespace: namespace}, role)
	return err
}

func getExpectedServiceMonitors(installType string) []string {
	if installType == string(integreatlyv1alpha1.InstallationTypeManaged3scale) {
		return commonExpectedServiceMonitors
	} else {
		return append(commonExpectedServiceMonitors, rhmi2ExpectedServiceMonitors...)
	}
}
