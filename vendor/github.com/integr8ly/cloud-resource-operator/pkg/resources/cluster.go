package resources

import (
	"bytes"
	"context"
	v1 "github.com/integr8ly/cloud-resource-operator/pkg/apis/config/v1"
	"io"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	errorUtil "github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetClusterID(ctx context.Context, c client.Client) (string, error) {
	infra := &v1.Infrastructure{}
	if err := c.Get(ctx, types.NamespacedName{Name: "cluster"}, infra); err != nil {
		return "", errorUtil.Wrap(err, "failed to retrieve cluster infrastructure")
	}
	return infra.Status.InfrastructureName, nil
}

func GetAWSRegion(ctx context.Context, c client.Client) (string, error) {
	infra, err := GetClusterInfrastructure(ctx, c)
	if err != nil {
		return "", errorUtil.Wrapf(err, "failure happened while retrieving cluster infrastructure")
	}
	if infra.Status.PlatformStatus.Type == v1.AWSPlatformType {
		return infra.Status.PlatformStatus.AWS.Region, nil
	}
	return "", errorUtil.New("infrastructure does not container aws region")
}

func GetClusterInfrastructure(ctx context.Context, c client.Client) (*v1.Infrastructure, error) {
	infra := &v1.Infrastructure{}
	if err := c.Get(ctx, types.NamespacedName{Name: "cluster"}, infra); err != nil {
		return nil, errorUtil.Wrap(err, "failed to retrieve cluster infrastructure")
	}
	return infra, nil
}

//go:generate moq -out cluster_moq.go . PodCommander
type PodCommander interface {
	ExecIntoPod(dpl *appsv1.Deployment, cmd string) error
}

type OpenShiftPodCommander struct {
	ClientSet *kubernetes.Clientset
}

func (pc *OpenShiftPodCommander) ExecIntoPod(dpl *appsv1.Deployment, cmd string) error {
	toRun := []string{"/bin/bash", "-c", cmd}
	podName, err := getDeploymentPod(pc.ClientSet, dpl)
	if err != nil {
		return err
	}
	if _, stderr, err := runExec(pc.ClientSet, toRun, podName, dpl.Namespace); err != nil {
		return errorUtil.Wrapf(err, "failed to exec, %s", stderr)
	}
	return nil
}

// run exec command on pod
func runExec(cs *kubernetes.Clientset, command []string, pod, ns string) (string, string, error) {
	req := cs.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod).
		Namespace(ns).
		SubResource("exec")

	req.VersionedParams(&corev1.PodExecOptions{
		Command: command,
		Stdin:   false,
		Stdout:  true,
		Stderr:  true,
		TTY:     false,
	}, scheme.ParameterCodec)

	cfg, _ := config.GetConfig()
	exec, err := remotecommand.NewSPDYExecutor(cfg, "POST", req.URL())
	if err != nil {
		return "", "", errorUtil.Wrap(err, "error while creating executor")
	}

	var stdout, stderr bytes.Buffer
	var stdin io.Reader
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})
	if err != nil {
		return stdout.String(), stderr.String(), err
	}

	return stdout.String(), stderr.String(), nil
}

func getDeploymentPod(cl *kubernetes.Clientset, dpl *appsv1.Deployment) (podName string, err error) {
	name := dpl.Name
	ns := dpl.Namespace
	api := cl.CoreV1()
	listOptions := metav1.ListOptions{
		LabelSelector: "deployment=" + name,
	}
	podList, _ := api.Pods(ns).List(context.Background(), listOptions)
	podListItems := podList.Items
	if len(podListItems) == 0 {
		return "", err
	}
	podName = podListItems[0].Name
	return podName, nil
}

func GetK8Client() (*kubernetes.Clientset, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(cfg)
}
