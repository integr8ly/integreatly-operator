package common

import (
	"bytes"
	goctx "context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"strings"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/remotecommand"
)

func execToPod(command string, podName string, namespace string, container string, ctx *TestingContext) (string, error) {
	req := ctx.KubeClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		Param("container", container)
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		return "", fmt.Errorf("error adding to scheme: %v", err)
	}
	parameterCodec := runtime.NewParameterCodec(scheme)
	req.VersionedParams(&corev1.PodExecOptions{
		Container: container,
		Command:   strings.Fields(command),
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}, parameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(ctx.KubeConfig, "POST", req.URL())
	if err != nil {
		return "", fmt.Errorf("error while creating Executor: %v", err)
	}

	var stdout, stderr bytes.Buffer
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})
	if err != nil {
		return "", fmt.Errorf("error in Stream: %v", err)
	}
	return stdout.String(), nil
}

// difference one-way diff that return strings in sliceSource that are not in sliceTarget
func difference(sliceSource, sliceTarget []string) []string {
	// create an empty lookup map with keys from sliceTarget
	diffSourceLookupMap := make(map[string]struct{}, len(sliceTarget))
	for _, item := range sliceTarget {
		diffSourceLookupMap[item] = struct{}{}
	}
	// use the lookup map to find items in sliceSource that are not in sliceTarget
	// and store them in a diff slice
	var diff []string
	for _, item := range sliceSource {
		if _, found := diffSourceLookupMap[item]; !found {
			diff = append(diff, item)
		}
	}
	return diff
}

// Is the cluster using on cluster or external storage
func isClusterStorage(ctx *TestingContext) (bool, error) {
	rhmi := &integreatlyv1alpha1.RHMI{}

	// get the RHMI custom resource to check what storage type is being used
	err := ctx.Client.Get(goctx.TODO(), types.NamespacedName{Name: InstallationName, Namespace: rhmiOperatorNamespace}, rhmi)
	if err != nil {
		return true, fmt.Errorf("error getting RHMI CR: %v", err)
	}

	if rhmi.Spec.UseClusterStorage == "true" {
		return true, nil
	}
	return false, nil
}
func FindTag(key, value string, tags []*rds.Tag) *rds.Tag {
	for _, tag := range tags {
		if key == aws.StringValue(tag.Key) && value == aws.StringValue(tag.Value) {
			return tag
		}
	}
	return nil
}

// arrayContains to return boolean flag if an arbitrary specified element is an element of that slice
func arrayContains(arr []string, targetValue string) bool {
	for _, element := range arr {
		if element != "" && element == targetValue {
			return true
		}
	}
	return false
}

//WrapLog Wrap an existing error with a message and log the provided message
func WrapLog(err error, msg string, logger *logrus.Entry) error {
	logger.Error(msg)
	return wrap(err, msg)
}
func wrap(err error, msg string) error {
	return fmt.Errorf("%s: %w", msg, err)
}
