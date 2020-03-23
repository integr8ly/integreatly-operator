package common

import (
	goctx "context"
	"fmt"
	"strings"
	"testing"

	"github.com/integr8ly/integreatly-operator/test/resources"
	projectv1 "github.com/openshift/api/project/v1"
	routev1 "github.com/openshift/api/route/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	projectName         = "project-test-e2e"
	serviceName         = "testing-curl"
	podName             = "testing-curl"
	containerName       = "testing-curl"
	threescaleNamespace = NamespacePrefix + "3scale"
	podEndpoitResponse  = "success"
)

func TestNetworkPolicyAccessNSToSVC(t *testing.T, ctx *TestingContext) {

	// ensure testing idp exists
	if !hasTestingIDP(ctx) {
		if err := setupTestingIDP(); err != nil {
			t.Fatalf("error setting up testing idp: %v", err)
		}
	}

	rhmi, err := getRHMI(ctx)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}

	// console master url
	masterURL := rhmi.Spec.MasterURL
	openshiftAPIURL := strings.Replace(rhmi.Spec.RoutingSubdomain, "apps.", "api.", 1) + ":6443"

	// getting oauth token as dedicated-admin
	token, err := getOauthToken(ctx, masterURL, "customer-admin01", "Password1")
	if err != nil {
		t.Fatalf("error authenticate user: %v", err)
	}

	// creating a project
	_, err = createProject(ctx, openshiftAPIURL, token)
	if err != nil {
		t.Fatalf("error occured while creating a project: %v", err)
	}

	// creating service
	serviceCR, err := createService(ctx, openshiftAPIURL, token)
	if err != nil {
		t.Fatalf("error occured while creating a service: %v", err)
	}

	// creating pod
	podCR, err := createPodWithAnEndpoint(ctx, openshiftAPIURL, token)
	if err != nil {
		t.Fatalf("error occured while creating a pod: %v", err)
	}

	podReady, err := checkPodStatus(ctx, podCR)
	if err != nil {
		t.Fatalf("error checking pod status: %v", err)
	}

	if podReady == false {
		t.Fatalf("pod %s failed to become ready", podCR.GetName())
	}

	tsApicastPod, err := get3ScaleAPIcastPod(ctx)
	if err != nil {
		t.Fatalf("error getting 3scale apicast pod: %v", err)
	}

	curlCommand := fmt.Sprintf("curl %s.%s.svc.cluster.local", serviceCR.GetName(), serviceCR.GetNamespace())
	apicastContainerName := "apicast-production"
	outputCurlCommand, err := execToPod(curlCommand,
		tsApicastPod.GetName(),
		tsApicastPod.GetNamespace(),
		apicastContainerName,
		ctx)
	if err != nil {
		t.Fatalf("error occured while executing curl command in pod: %v", err)
	}

	if strings.TrimSpace(outputCurlCommand) != podEndpoitResponse {
		t.Fatalf("Failed to validate response Pod apicast received %s, instead of %s", outputCurlCommand, podEndpoitResponse)
	}

	cleanUp(ctx)

}

func getOauthToken(ctx *TestingContext, apiUrl string, username string, password string) (string, error) {

	// get oauth route
	oauthRoute := &routev1.Route{}
	key := types.NamespacedName{
		Name:      resources.OpenshiftOAuthRouteName,
		Namespace: resources.OpenshiftAuthenticationNamespace,
	}

	if err := ctx.Client.Get(goctx.TODO(), key, oauthRoute); err != nil {
		return "", fmt.Errorf("error getting Openshift Oauth Route: %s", err)
	}

	// get dedicated admin token
	dedicatedAdminToken, err := resources.DoAuthOpenshiftUser(
		oauthRoute.Spec.Host,
		apiUrl,
		resources.DefaultIDP,
		username,
		password,
	)
	if err != nil {
		return "", fmt.Errorf("error occured trying to get token : %v", err)
	}

	return dedicatedAdminToken, nil
}

// creating project as dedicated admin
func createProject(ctx *TestingContext, openshiftAPIURL string, token string) (*projectv1.ProjectRequest, error) {
	projectCR := &projectv1.ProjectRequest{
		ObjectMeta: v1.ObjectMeta{
			Name: projectName,
		},
	}

	// check if project exist
	err := ctx.Client.Get(goctx.TODO(), k8sclient.ObjectKey{Name: projectName}, &projectv1.Project{})
	if err != nil && !k8serr.IsNotFound(err) {
		return nil, fmt.Errorf("error occured while retrieving a project: %v", err)
	} else if k8serr.IsNotFound(err) {
		if err := resources.DoOpenshiftCreateProject(openshiftAPIURL, token, projectCR); err != nil {
			return nil, fmt.Errorf("error occured while making request to create a project: %v", err)
		}
	}

	return projectCR, nil
}

// creates service as dedicated-admin
func createService(ctx *TestingContext, openshiftAPIURL string, token string) (*corev1.Service, error) {
	serviceCR := &corev1.Service{
		ObjectMeta: v1.ObjectMeta{
			Name:      serviceName,
			Namespace: projectName,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				corev1.ServicePort{
					Protocol:   corev1.ProtocolTCP,
					Port:       80,
					TargetPort: intstr.FromInt(8080),
				},
			},
			Selector: getTestingCurlLabels(),
		},
	}

	// check if service exist
	key := k8sclient.ObjectKey{Name: serviceCR.GetName(), Namespace: serviceCR.GetNamespace()}
	err := ctx.Client.Get(goctx.TODO(), key, serviceCR)
	if err != nil && !k8serr.IsNotFound(err) {
		return nil, fmt.Errorf("error occured while retrieving service %s : %v", serviceCR.GetName(), err)
	} else if k8serr.IsNotFound(err) {
		err = resources.DoOpenshiftCreateServiceInANamespace(openshiftAPIURL, token, projectName, serviceCR)
		if err != nil {
			return nil, fmt.Errorf("error occured while making request to create service %s : %v", serviceCR.GetName(), err)
		}
	}

	return serviceCR, nil
}

func createPodWithAnEndpoint(ctx *TestingContext, openshiftAPIURL string, token string) (*corev1.Pod, error) {

	podCR := &corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name:      podName,
			Labels:    getTestingCurlLabels(),
			Namespace: projectName,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				corev1.Container{
					Name:  containerName,
					Image: "busybox:1.31.1",
					Command: []string{
						"/bin/sh",
					},
					Args: []string{
						"-c",
						fmt.Sprintf("while true ; do  echo -e \"HTTP/1.1 200 OK\n\n %s \" | nc -l -p 8080  ; done", podEndpoitResponse),
					},
					Ports: []corev1.ContainerPort{
						corev1.ContainerPort{
							ContainerPort: 8080,
						},
					},
				},
			},
		},
	}

	// check if pod exist
	key := k8sclient.ObjectKey{Name: podCR.GetName(), Namespace: podCR.GetNamespace()}
	err := ctx.Client.Get(goctx.TODO(), key, podCR)
	if err != nil && !k8serr.IsNotFound(err) {
		return nil, fmt.Errorf("error occured while retrieving pod %s : %v", podCR.GetName(), err)
	} else if k8serr.IsNotFound(err) {
		err = resources.DoOpenshiftCreatePodInANamespace(openshiftAPIURL, token, projectName, podCR)
		if err != nil {
			return nil, fmt.Errorf("error occured while making request to create pod %s : %v", podCR.GetName(), err)
		}
	}

	return podCR, nil

}

func checkPodStatus(ctx *TestingContext, podCR *corev1.Pod) (bool, error) {
	checkPodStatus := false
	key := k8sclient.ObjectKey{Name: podCR.GetName(), Namespace: podCR.GetNamespace()}
	for !checkPodStatus {
		err := ctx.Client.Get(goctx.TODO(), key, podCR)
		if err != nil {
			return false, fmt.Errorf("error getting pod: %v", err)
		}
		for _, cnd := range podCR.Status.Conditions {
			if cnd.Type == corev1.ContainersReady {
				if cnd.Status != corev1.ConditionStatus("True") {
					checkPodStatus = false
				} else {
					checkPodStatus = true
					break
				}
			}
		}
	}
	return checkPodStatus, nil
}

func get3ScaleAPIcastPod(ctx *TestingContext) (*corev1.Pod, error) {

	listOptions := []k8sclient.ListOption{
		k8sclient.MatchingLabels(map[string]string{
			"threescale_component": "apicast",
		}),
		k8sclient.InNamespace(threescaleNamespace),
	}

	tsApicastPods := &corev1.PodList{}

	err := ctx.Client.List(goctx.TODO(), tsApicastPods, listOptions...)
	if err != nil {
		return nil, fmt.Errorf("error listing 3scale apicast pods: %v", err)
	}

	tsApicastPod := tsApicastPods.Items[0]
	return &tsApicastPod, nil
}

func getTestingCurlLabels() map[string]string {
	return map[string]string{
		"app": "testing-curl",
	}
}

func cleanUp(ctx *TestingContext) error {
	project := &projectv1.Project{ObjectMeta: v1.ObjectMeta{Name: projectName}}
	service := &corev1.Service{ObjectMeta: v1.ObjectMeta{Name: serviceName, Namespace: projectName}}
	pod := &corev1.Pod{ObjectMeta: v1.ObjectMeta{Name: podName, Namespace: projectName}}

	models := []runtime.Object{pod, service, project}
	for _, model := range models {
		if err := ctx.Client.Delete(goctx.TODO(), model); err != nil {
			return err
		}
	}

	return nil
}
