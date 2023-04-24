package marin3r

import (
	"encoding/json"
	"fmt"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
)

type LimitadorClientInterface interface {
	GetLimitsByName(string) ([]limitadorLimit, error)
	CurlDeleteLimitsByNameUsingPod(string, string, string, string) error
}

type LimitadorClient struct {
	PodExecutor resources.PodExecutorInterface
	PodName     string
	Namespace   string
}

var _ LimitadorClientInterface = &LimitadorClient{}

func NewLimitadorClient(podExecutor resources.PodExecutorInterface, nameSpace, podName string) *LimitadorClient {
	return &LimitadorClient{
		PodExecutor: podExecutor,
		Namespace:   nameSpace,
		PodName:     podName,
	}
}

func (l LimitadorClient) GetLimitsByName(limitName string) ([]limitadorLimit, error) {
	response, _, err := l.PodExecutor.ExecuteRemoteCommand(l.Namespace, l.PodName, []string{"/bin/sh",
		"-c", fmt.Sprintf("wget -qO - http://127.0.0.1:8080/limits/%s", limitName)})
	if err != nil {
		return nil, err
	}

	limitadorLimitsInRedis := []limitadorLimit{}
	err = json.Unmarshal([]byte(response), &limitadorLimitsInRedis)
	if err != nil {
		return nil, err
	}

	return limitadorLimitsInRedis, nil
}

func (l LimitadorClient) CurlDeleteLimitsByNameUsingPod(limitName, namespace, podName, rateLimitPodIP string) error {
	_, _, err := l.PodExecutor.ExecuteRemoteCommand(namespace, podName, []string{"/bin/sh",
		"-c", fmt.Sprintf("curl -X DELETE %s:8080/limits/%s", rateLimitPodIP, limitName)})
	if err != nil {
		return err
	}

	return nil
}
