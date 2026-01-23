package resources

import (
	"context"
	errorUtil "github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	TagDisplayName = "Name"

	TagManagedKey = "red-hat-managed"
	TagManagedVal = "true"
)

// generic key-value tag
type Tag struct {
	Key   string
	Value string
}

func (a *Tag) Equal(b *Tag) bool {
	return a.Key == b.Key && a.Value == b.Value
}

// MergeTags merges generalTags and infraTags, where any duplicate key in
// infraTags is discarded in favour of the value in infraTags
func MergeTags(generalTags []*Tag, infraTags []*Tag) []*Tag {
	var dupMap = make(map[string]bool)
	for _, tag := range generalTags {
		dupMap[tag.Key] = true
	}
	for _, tag := range infraTags {
		if _, exists := dupMap[tag.Key]; !exists {
			generalTags = append(generalTags, tag)
		}
	}
	return generalTags
}

func TagsContains(tags []*Tag, key, value string) bool {
	for _, tag := range tags {
		if tag.Key == key && tag.Value == value {
			return true
		}
	}
	return false
}

// TagsContainsAll checks whether all tags in first parameter are contained within second parameter
func TagsContainsAll(as []*Tag, bs []*Tag) bool {
	for _, a := range as {
		found := false
		for _, b := range bs {
			if a.Equal(b) {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func GetDefaultResourceTags(ctx context.Context, c client.Client, specType string, name string, prodName string) ([]*Tag, string, error) {
	// set the tag values that will always be added
	defaultOrganizationTag := GetOrganizationTag()
	clusterID, err := GetClusterID(ctx, c)
	if err != nil {
		msg := "Failed to get cluster id"
		return nil, "", errorUtil.Wrap(err, msg)
	}
	tags := []*Tag{
		{
			Key:   defaultOrganizationTag + "clusterID",
			Value: clusterID,
		},
		{
			Key:   defaultOrganizationTag + "resource-type",
			Value: specType,
		},
		{
			Key:   defaultOrganizationTag + "resource-name",
			Value: name,
		},
		BuildManagedTag(),
	}

	if prodName != "" {
		productTag := &Tag{
			Key:   defaultOrganizationTag + "product-name",
			Value: prodName,
		}
		tags = append(tags, productTag)
	}

	infraTags, err := GetUserInfraTags(ctx, c)
	if err != nil {
		msg := "Failed to get user infrastructure tags"
		return nil, "", errorUtil.Wrap(err, msg)
	}
	if infraTags != nil {
		// merge tags into single array, where any duplicate
		// values in infra are overwritten by the default tags
		tags = MergeTags(infraTags, tags)
	}

	return tags, clusterID, nil
}

func GetUserInfraTags(ctx context.Context, c client.Client) ([]*Tag, error) {
	// get infra CR
	infra, err := GetClusterInfrastructure(ctx, c)
	if err != nil {
		msg := "failed to get cluster infrastructure"
		return nil, errorUtil.Wrap(err, msg)
	}

	var tags []*Tag
	// retrieve all user infrastructure tags
	if infra.Status.PlatformStatus != nil && infra.Status.PlatformStatus.AWS != nil {
		for _, t := range infra.Status.PlatformStatus.AWS.ResourceTags {
			tags = append(tags, &Tag{Key: t.Key, Value: t.Value})
		}
	}
	return tags, nil
}

func BuildManagedTag() *Tag {
	return &Tag{
		Key:   TagManagedKey,
		Value: TagManagedVal,
	}
}
