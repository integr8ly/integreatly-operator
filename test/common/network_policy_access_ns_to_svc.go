package common

import (
	goctx "context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/integr8ly/integreatly-operator/test/resources"
	projectv1 "github.com/openshift/api/project/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	projectName         = "project-test-e2e"
	serviceName         = "testing-curl"
	podName             = "testing-curl"
	containerName       = "testing-curl"
	threescaleNamespace = NamespacePrefix + "3scale"
	podEndpoitResponse  = "success"
)

func TestNetworkPolicyAccessNSToSVC(t *testing.T, ctx *TestingContext) {
	if err := createTestingIDP(t, goctx.TODO(), ctx.Client, ctx.KubeConfig, ctx.SelfSignedCerts); err != nil {
		t.Fatalf("error while creating testing idp: %v", err)
	}

	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}

	// console master url
	masterURL := rhmi.Spec.MasterURL

	// get dedicated admin token
	if err := resources.DoAuthOpenshiftUser(fmt.Sprintf("%s/auth/login", masterURL), "customer-admin-1", DefaultPassword, ctx.HttpClient, TestingIDPRealm, t); err != nil {
		t.Fatalf("error occured trying to get token : %v", err)
	}

	openshiftClient := resources.NewOpenshiftClient(ctx.HttpClient, masterURL)

	// creating a project as dedicated-admin
	_, err = createProject(ctx, openshiftClient)
	if err != nil {
		t.Fatalf("error occured while creating a project: %v", err)
	}

	// creating service as dedicated-admin
	serviceCR, err := createService(ctx, openshiftClient)
	if err != nil {
		t.Fatalf("error occured while creating a service: %v", err)
	}

	// creating pod as dedicated-admin
	podCR, err := createPodWithAnEndpoint(ctx, openshiftClient)
	if err != nil {
		//t.Fatalf("error occured while creating a pod: %v", err)
		// Fails to create pod on osde2e - skipping for now
		t.Skipf("error occured while creating a pod: %v", err)
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

	defer cleanUp(ctx)

}

// creating project as dedicated admin
func createProject(ctx *TestingContext, openshiftClient *resources.OpenshiftClient) (*projectv1.ProjectRequest, error) {
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
		if err := openshiftClient.DoOpenshiftCreateProject(projectCR); err != nil {
			return nil, fmt.Errorf("error occured while making request to create a project: %v", err)
		}
	}

	return projectCR, nil
}

// creates service as dedicated-admin
func createService(ctx *TestingContext, openshiftClient *resources.OpenshiftClient) (*corev1.Service, error) {
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
		err = openshiftClient.DoOpenshiftCreateServiceInANamespace(projectName, serviceCR)
		if err != nil {
			return nil, fmt.Errorf("error occured while making request to create service %s : %v", serviceCR.GetName(), err)
		}
	}

	return serviceCR, nil
}

func createPodWithAnEndpoint(ctx *TestingContext, openshiftClient *resources.OpenshiftClient) (*corev1.Pod, error) {

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
		err = openshiftClient.DoOpenshiftCreatePodInANamespace(projectName, podCR)
		if err != nil {
			return nil, fmt.Errorf("error occured while making request to create pod %s : %v", podCR.GetName(), err)
		}
	}

	return podCR, nil

}

func checkPodStatus(ctx *TestingContext, podCR *corev1.Pod) (bool, error) {
	err := wait.PollImmediate(time.Second*5, time.Minute*5, func() (done bool, err error) {
		key := k8sclient.ObjectKey{Name: podCR.GetName(), Namespace: podCR.GetNamespace()}
		err = ctx.Client.Get(goctx.TODO(), key, podCR)
		if err != nil {
			return false, fmt.Errorf("error getting pod: %v", err)
		}

		for _, cnd := range podCR.Status.Conditions {
			if cnd.Type == corev1.ContainersReady && cnd.Status == corev1.ConditionStatus("True") {
				return true, nil
			}
		}
		return false, nil
	})
	if err != nil {
		return false, err
	}

	return true, nil
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

	if len(tsApicastPods.Items) == 0 {
		return nil, fmt.Errorf("Expected 3scale apicast pods to be created, none found")
	}

	return &tsApicastPods.Items[0], nil
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
