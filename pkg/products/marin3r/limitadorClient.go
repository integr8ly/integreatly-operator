package marin3r

import (
	"encoding/json"
	"fmt"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"net/http"
	"time"
)

type LimitadorClientInterface interface {
	GetLimitsByName(string) ([]limitadorLimit, error)
	DeleteLimitsByNameUsingPod(string, string) error
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
		"-c", fmt.Sprintf("curl -fsSL http://127.0.0.1:8080/limits/%s", limitName)})
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

func (l LimitadorClient) DeleteLimitsByNameUsingPod(limitName, rateLimitPodIP string) error {
	client := &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
			IdleConnTimeout:   10 * time.Second,
		},
		Timeout: 10 * time.Second,
	}
	url := fmt.Sprintf("http://%s:8080/limits/%s", rateLimitPodIP, limitName)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	_, err = client.Do(req)
	if err != nil {
		return err
	}

	return nil
}
