package ratelimit

import (
	"context"
	"fmt"
	"strconv"

	"github.com/3scale-ops/marin3r/pkg/envoy"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	appsv1 "github.com/openshift/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	k8sTypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	EnvoyImage = "registry.redhat.io/openshift-service-mesh/proxyv2-rhel8:2.5.3-8"
)

type envoyProxyServer struct {
	ctx    context.Context
	client k8sclient.Client
	log    l.Logger
}

func NewEnvoyProxyServer(ctx context.Context, client k8sclient.Client, logger l.Logger) *envoyProxyServer {
	return &envoyProxyServer{
		ctx:    ctx,
		client: client,
		log:    logger,
	}
}

func (envoyProxy *envoyProxyServer) CreateEnvoyProxyContainer(dcName, namespace, envoyNodeID, svcProxyName, svcProxyPortName string, svcProxyPort int) (integreatlyv1alpha1.StatusPhase, error) {

	envoyProxy.log.Infof(
		"Creating envoy sidecar container for: ",
		l.Fields{"DeploymentConfig": dcName, "Namespace": namespace},
	)

	// patches deployment config to add the sidecar container
	phase, err := envoyProxy.patchDeploymentConfig(dcName, namespace, envoyNodeID, svcProxyPort)
	if err != nil {
		return phase, err
	}

	// forwards request to the envoy proxy container
	phase, err = envoyProxy.patchService(svcProxyName, namespace, svcProxyPortName, svcProxyPort)
	if err != nil {
		return phase, err
	}

	return phase, nil
}

func (envoyProxy *envoyProxyServer) patchDeploymentConfig(dcName, namespace, envoyNodeID string, svcProxyPort int) (integreatlyv1alpha1.StatusPhase, error) {

	dc, phase, err := getDeploymentConfig(envoyProxy.ctx, envoyProxy.client, dcName, namespace)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to get %s deploymentconfig on namespace %s : %w", dcName, namespace, err)
	}
	if phase == integreatlyv1alpha1.PhaseAwaitingComponents {
		envoyProxy.log.Infof(
			"Waiting for deploymentconfig to be available",
			l.Fields{"DeploymentConfig": dcName},
		)
		return phase, nil
	}

	if dc.Spec.Template.Labels == nil {
		dc.Spec.Template.SetLabels(make(map[string]string))
	}
	if dc.Spec.Template.Annotations == nil {
		dc.Spec.Template.SetAnnotations(make(map[string]string))
	}

	envoyPort := fmt.Sprintf("envoy-https:%s", strconv.Itoa(svcProxyPort))

	envoyProxy.log.Infof(
		"adding MARIN3R annotations and labels: ", l.Fields{
			"marin3r.3scale.net/node-id":           envoyNodeID,
			"marin3r.3scale.net/ports":             envoyPort,
			"marin3r.3scale.net/envoy-image":       EnvoyImage,
			"marin3r.3scale.net/status":            "enabled",
			"marin3r.3scale.net/envoy-api-version": envoy.APIv3.String(),
		})

	dc.Spec.Template.Labels["marin3r.3scale.net/status"] = "enabled"
	dc.Spec.Template.Annotations["marin3r.3scale.net/node-id"] = envoyNodeID
	dc.Spec.Template.Annotations["marin3r.3scale.net/ports"] = envoyPort
	dc.Spec.Template.Annotations["marin3r.3scale.net/envoy-api-version"] = envoy.APIv3.String()
	dc.Spec.Template.Annotations["marin3r.3scale.net/envoy-image"] = EnvoyImage
	dc.Spec.Template.Annotations["marin3r.3scale.net/resources.requests.cpu"] = "190m"
	dc.Spec.Template.Annotations["marin3r.3scale.net/resources.requests.memory"] = "90Mi"

	if err := envoyProxy.client.Update(envoyProxy.ctx, dc); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to apply MARIN3R labels to %s deploymentconfig: %v", dcName, err)
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (envoyProxy *envoyProxyServer) patchService(svcName, namespace, portName string, svcPort int) (integreatlyv1alpha1.StatusPhase, error) {

	envoyProxy.log.Infof(
		"Patching service to point to proxy service",
		l.Fields{"Service": svcName, "ServicePort": svcPort},
	)

	service, phase, err := getService(envoyProxy.ctx, envoyProxy.client, svcName, namespace)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	if phase == integreatlyv1alpha1.PhaseAwaitingComponents {
		envoyProxy.log.Infof(
			"Waiting for service to be available",
			l.Fields{"Service": svcName},
		)
		return phase, nil
	}

	ports := service.Spec.Ports
	for i, port := range ports {
		if port.Name == portName {
			service.Spec.Ports[i].Port = int32(svcPort)
			service.Spec.Ports[i].TargetPort = intstr.FromInt(svcPort)
			break
		}
	}
	if err := envoyProxy.client.Update(envoyProxy.ctx, service); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to update %s service to forward requests to proxy server: %v", svcName, err)
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func getDeploymentConfig(ctx context.Context, client k8sclient.Client, dcName string, dcNamespace string) (*appsv1.DeploymentConfig, integreatlyv1alpha1.StatusPhase, error) {
	apiCastDeploymentConfig := &appsv1.DeploymentConfig{}

	err := client.Get(ctx, k8sTypes.NamespacedName{Name: dcName, Namespace: dcNamespace}, apiCastDeploymentConfig)

	if err != nil {
		if k8serr.IsNotFound(err) {
			return nil, integreatlyv1alpha1.PhaseAwaitingComponents, nil
		}
		return nil, integreatlyv1alpha1.PhaseFailed, err
	}
	return apiCastDeploymentConfig, integreatlyv1alpha1.PhaseInProgress, nil
}

func getService(ctx context.Context, client k8sclient.Client, svcName string, svcNamespace string) (*corev1.Service, integreatlyv1alpha1.StatusPhase, error) {
	service := &corev1.Service{}
	err := client.Get(ctx, k8sTypes.NamespacedName{Name: svcName, Namespace: svcNamespace}, service)
	if err != nil {
		if k8serr.IsNotFound(err) {
			return nil, integreatlyv1alpha1.PhaseAwaitingComponents, nil
		}
		return nil, integreatlyv1alpha1.PhaseFailed, err
	}
	return service, integreatlyv1alpha1.PhaseInProgress, nil
}
