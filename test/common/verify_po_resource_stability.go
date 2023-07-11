package common

import (
	goctx "context"
	"fmt"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

var (
	serviceMonitorNameNamespace = ThreeScaleOperatorNamespace
	serviceMonitorName          = "threescale-operator-controller-manager-metrics-monitor"
)

func TestPackageOperatorResourceStability(t TestingTB, ctx *TestingContext) {
	// Fetch the initial state of the resource
	resource, err := getServiceMonitor(ctx, serviceMonitorNameNamespace, serviceMonitorName)
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

func getServiceMonitor(ctx *TestingContext, namespace, serviceName string) (*monitoringv1.ServiceMonitor, error) {
	sm := &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
		},
	}
	err := ctx.Client.Get(goctx.TODO(), k8sclient.ObjectKey{Name: sm.Name, Namespace: sm.Namespace}, sm)
	if err != nil {
		return nil, fmt.Errorf("ServiceMonitor %s not found in namespace %s", serviceName, namespace)
	}

	return sm, nil
}
