package resources

import (
	"bytes"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	kube "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

// ExecCmd exec command on specific pod and wait the command's output.
//func ExecuteRemoteCommand(ns string, podName string, command string, container string) (string, string, error) {
func ExecuteRemoteCommand(ns string, podName string, command string) (string, string, error) {
	cmd := []string{
		"/bin/bash",
		"-c",
		command,
	}

	kubeClient, restConfig, err := getClient()
	if err != nil {
		return "", "", errors.Wrapf(err, "Failed to get client")
	}

	req := kubeClient.CoreV1().RESTClient().Post().Resource("pods").Name(podName).
		Namespace(ns).SubResource("exec")
	option := &v1.PodExecOptions{
		Command: cmd,
		Stdin:   false,
		Stdout:  true,
		Stderr:  true,
		TTY:     true,
		//Container: container,
	}
	req.VersionedParams(
		option,
		scheme.ParameterCodec,
	)
	exec, err := remotecommand.NewSPDYExecutor(restConfig, "POST", req.URL())
	if err != nil {
		return "", "", errors.Wrapf(err, "Failed executing command %s on %s/%s", command, ns, podName)
	}

	buf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}

	logrus.Infof("Executing command, %s, on pod, %s", command, podName)

	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: buf,
		Stderr: errBuf,
	})
	if err != nil {
		return "", "", errors.Wrapf(err, "Failed executing command %s on %s/%s", command, ns, podName)
	}

	return buf.String(), errBuf.String(), nil
}

func getClient() (*kube.Clientset, *restclient.Config, error) {

	kubeCfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)
	restCfg, err := kubeCfg.ClientConfig()

	kubeClient, err := kube.NewForConfig(restCfg)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "Failed to generate new client")
	}
	return kubeClient, restCfg, nil
}
