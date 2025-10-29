package ratelimit

import (
	"context"
	"fmt"
	"strconv"

	"github.com/3scale-ops/marin3r/pkg/envoy"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	k8sTypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	EnvoyImage = "registry.redhat.io/openshift-service-mesh/proxyv2-rhel9:2.6.9-6"
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

func (envoyProxy *envoyProxyServer) CreateEnvoyProxyContainer(deploymentName, namespace, envoyNodeID, svcProxyName, svcProxyPortName string, svcProxyPort int) (integreatlyv1alpha1.StatusPhase, error) {

	envoyProxy.log.Infof(
		"Creating envoy sidecar container for: ",
		l.Fields{"Deployment": deploymentName, "Namespace": namespace},
	)

	// patches deployment to add the sidecar container
	phase, err := envoyProxy.patchDeployment(deploymentName, namespace, envoyNodeID, svcProxyPort)
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

func (envoyProxy *envoyProxyServer) patchDeployment(deploymentName, namespace, envoyNodeID string, svcProxyPort int) (integreatlyv1alpha1.StatusPhase, error) {

	deployment, phase, err := getDeployment(envoyProxy.ctx, envoyProxy.client, deploymentName, namespace)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to get %s deployment on namespace %s : %w", deploymentName, namespace, err)
	}
	if phase == integreatlyv1alpha1.PhaseAwaitingComponents {
		envoyProxy.log.Infof(
			"Waiting for deployment to be available",
			l.Fields{"Deployment": deploymentName},
		)
		return phase, nil
	}

	if deployment.Spec.Template.Labels == nil {
		deployment.Spec.Template.SetLabels(make(map[string]string))
	}
	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.SetAnnotations(make(map[string]string))
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

	deployment.Spec.Template.Labels["marin3r.3scale.net/status"] = "enabled"
	deployment.Spec.Template.Annotations["marin3r.3scale.net/node-id"] = envoyNodeID
	deployment.Spec.Template.Annotations["marin3r.3scale.net/ports"] = envoyPort
	deployment.Spec.Template.Annotations["marin3r.3scale.net/envoy-api-version"] = envoy.APIv3.String()
	deployment.Spec.Template.Annotations["marin3r.3scale.net/envoy-image"] = EnvoyImage
	deployment.Spec.Template.Annotations["marin3r.3scale.net/resources.requests.cpu"] = "190m"
	deployment.Spec.Template.Annotations["marin3r.3scale.net/resources.requests.memory"] = "90Mi"

	if err := envoyProxy.client.Update(envoyProxy.ctx, deployment); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to apply MARIN3R labels to %s deployment: %v", deployment, err)
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
			// #nosec G115 -- Port numbers are guaranteed to be within the valid range.
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

func getDeployment(ctx context.Context, client k8sclient.Client, deploymentName string, deploymentNamespace string) (*appsv1.Deployment, integreatlyv1alpha1.StatusPhase, error) {
	apiCastDeployment := &appsv1.Deployment{}

	err := client.Get(ctx, k8sTypes.NamespacedName{Name: deploymentName, Namespace: deploymentNamespace}, apiCastDeployment)

	if err != nil {
		if k8serr.IsNotFound(err) {
			return nil, integreatlyv1alpha1.PhaseAwaitingComponents, nil
		}
		return nil, integreatlyv1alpha1.PhaseFailed, err
	}
	return apiCastDeployment, integreatlyv1alpha1.PhaseInProgress, nil
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
