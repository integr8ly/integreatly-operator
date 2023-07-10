package common

import (
	goctx "context"
	"fmt"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

var (
	threescaleOperatorNs = "redhat-rhoam-3scale-operator"
	serviceMonitorName   = "threescale-operator-controller-manager-metrics-monitor"
)

func TestPackageOperatorResourceStability(t TestingTB, ctx *TestingContext) {
	// Fetch the initial state of the resource
	resource, err := getServiceMonitor(ctx, threescaleOperatorNs, serviceMonitorName)
	if err != nil {
		t.Fatalf("Failed to fetch resource: %v", err)
	}

	t.Log("Waiting 30 seconds")
	time.Sleep(30 * time.Second)

	// Update resource, will fail if resource has been modified after fetch
	err = ctx.Client.Update(goctx.TODO(), resource)
	if err != nil {
		t.Fatalf("Failed to update resource: %v", err)
	}

}

func getServiceMonitor(ctx *TestingContext, nameSpace, serviceName string) (*monitoringv1.ServiceMonitor, error) {
	// Get the list of service monitors in the namespace
	listOpts := []k8sclient.ListOption{
		k8sclient.InNamespace(nameSpace),
	}

	serviceMonitors := &monitoringv1.ServiceMonitorList{}
	err := ctx.Client.List(goctx.TODO(), serviceMonitors, listOpts...)
	if err != nil {
		return nil, err
	}

	// Iterate over the service monitors to find the one with the specified name
	for _, sm := range serviceMonitors.Items {
		if sm.Name == serviceName {
			return sm, nil
		}
	}

	// If the specific service monitor is not found, return an error
	return nil, fmt.Errorf("ServiceMonitor %s not found in namespace %s", serviceName, nameSpace)
}
