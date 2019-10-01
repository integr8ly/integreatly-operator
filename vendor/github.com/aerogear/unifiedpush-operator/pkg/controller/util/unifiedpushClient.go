package util

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aerogear/unifiedpush-operator/pkg/unifiedpush"
)

func UnifiedpushClient(kclient client.Client, reqLogger logr.Logger) (unifiedpush.UnifiedpushClient, error) {
	listOptions := client.MatchingLabels(map[string]string{"internal": "unifiedpush"})

	var foundServices corev1.ServiceList
	err := kclient.List(context.TODO(), listOptions, &foundServices)
	if err != nil {
		return unifiedpush.UnifiedpushClient{}, err
	}

	if len(foundServices.Items) == 0 {
		return unifiedpush.UnifiedpushClient{}, errors.New("Didn't find any UPS Services")
	}

	if len(foundServices.Items) > 1 {
		reqLogger.Info("Found more than one internal UPS Service, using the first one")
	}

	upsUrl := fmt.Sprintf("http://%s", foundServices.Items[0].Name)
	return unifiedpush.UnifiedpushClient{upsUrl}, nil
}
