package gcp

import (
	"context"
	"fmt"
	"github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1"
	"github.com/integr8ly/cloud-resource-operator/pkg/resources"
	errorUtil "github.com/pkg/errors"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

func buildDefaultRedisTags(ctx context.Context, client k8sclient.Client, r *v1alpha1.Redis) (map[string]string, error) {
	if r == nil {
		return nil, fmt.Errorf("cannot build default gcp redis tags because the redis cr is nil")
	}
	defaultTags, _, err := resources.GetDefaultResourceTags(ctx, client, r.Spec.Type, r.Name, r.ObjectMeta.Labels["productName"])
	if err != nil {
		return nil, errorUtil.Wrapf(err, "failed to get default redis tags")
	}
	tags := make(map[string]string, len(defaultTags))
	// GCP labels cannot have uppercase, dash or slash characters
	// ref: https://cloud.google.com/resource-manager/docs/creating-managing-labels#requirements
	replacer := strings.NewReplacer(".", "-", "/", "_")
	for _, tag := range defaultTags {
		key := strings.ToLower(replacer.Replace(tag.Key))
		value := strings.ToLower(replacer.Replace(tag.Value))
		tags[key] = value
	}
	return tags, nil
}

func buildDefaultPostgresTags(ctx context.Context, client k8sclient.Client, pg *v1alpha1.Postgres) (map[string]string, error) {
	defaultTags, _, err := resources.GetDefaultResourceTags(ctx, client, pg.Spec.Type, pg.Name, pg.ObjectMeta.Labels["productName"])
	if err != nil {
		return nil, errorUtil.Wrapf(err, "failed to get default postgres tags")
	}
	tags := make(map[string]string, len(defaultTags))
	// GCP labels cannot have uppercase, dash or slash characters
	// ref: https://cloud.google.com/resource-manager/docs/creating-managing-labels#requirements
	replacer := strings.NewReplacer(".", "-", "/", "_")
	for _, tag := range defaultTags {
		key := strings.ToLower(replacer.Replace(tag.Key))
		value := strings.ToLower(replacer.Replace(tag.Value))
		tags[key] = value
	}
	return tags, nil
}
