package aws

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/integr8ly/cloud-resource-operator/pkg/resources"
	errorUtil "github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	tagDisplayName = "Name"

	tagManagedKey = "red-hat-managed"
	tagManagedVal = "true"
)

// generic key-value tag
type tag struct {
	key   string
	value string
}

func (a *tag) Equal(b *tag) bool {
	return a.key == b.key && a.value == b.value
}

func ec2TagToGeneric(ec2Tag *ec2.Tag) *tag {
	return &tag{key: aws.StringValue(ec2Tag.Key), value: aws.StringValue(ec2Tag.Value)}
}

func ec2TagsToGeneric(ec2Tags []*ec2.Tag) []*tag {
	var genericTags []*tag
	for _, ec2Tag := range ec2Tags {
		genericTags = append(genericTags, &tag{key: aws.StringValue(ec2Tag.Key), value: aws.StringValue(ec2Tag.Value)})
	}
	return genericTags
}

func genericToEc2Tag(tag *tag) *ec2.Tag {
	return &ec2.Tag{Key: aws.String(tag.key), Value: aws.String(tag.value)}
}

func genericToRdsTag(tag *tag) *rds.Tag {
	return &rds.Tag{Key: aws.String(tag.key), Value: aws.String(tag.value)}
}

func genericToS3Tag(tag *tag) *s3.Tag {
	return &s3.Tag{Key: aws.String(tag.key), Value: aws.String(tag.value)}
}

func genericToElasticacheTag(tag *tag) *elasticache.Tag {
	return &elasticache.Tag{Key: aws.String(tag.key), Value: aws.String(tag.value)}
}

func genericToRdsTags(tags []*tag) []*rds.Tag {
	var rdsTags []*rds.Tag
	for _, tag := range tags {
		rdsTags = append(rdsTags, genericToRdsTag(tag))
	}
	return rdsTags
}

func genericToS3Tags(tags []*tag) []*s3.Tag {
	var s3Tags []*s3.Tag
	for _, tag := range tags {
		s3Tags = append(s3Tags, genericToS3Tag(tag))
	}
	return s3Tags
}

func genericToElasticacheTags(tags []*tag) []*elasticache.Tag {
	var cacheTags []*elasticache.Tag
	for _, tag := range tags {
		cacheTags = append(cacheTags, genericToElasticacheTag(tag))
	}
	return cacheTags
}

func rdsTagstoGeneric(rdsTags []*rds.Tag) []*tag {
	var genericTags []*tag
	for _, rdsTag := range rdsTags {
		genericTags = append(genericTags, &tag{key: aws.StringValue(rdsTag.Key), value: aws.StringValue(rdsTag.Value)})
	}
	return genericTags
}

func genericToEc2Tags(tags []*tag) []*ec2.Tag {
	var ec2Tags []*ec2.Tag
	for _, tag := range tags {
		ec2Tags = append(ec2Tags, genericToEc2Tag(tag))
	}
	return ec2Tags
}

// this function merges generalTags and infraTags, where any duplicate key in
// infraTags is discarded in favour of the value in infraTags
func mergeTags(generalTags []*tag, infraTags []*tag) []*tag {
	var dupMap = make(map[string]bool)
	for _, tag := range generalTags {
		dupMap[tag.key] = true
	}
	for _, tag := range infraTags {
		if _, exists := dupMap[tag.key]; !exists {
			generalTags = append(generalTags, tag)
		}
	}
	return generalTags
}

func tagsContains(tags []*tag, key, value string) bool {
	for _, tag := range tags {
		if tag.key == key && tag.value == value {
			return true
		}
	}
	return false
}

// Checks whether all tags in first parameter are contained within second parameter
func tagsContainsAll(as []*tag, bs []*tag) bool {
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

func getDefaultResourceTags(ctx context.Context, c client.Client, specType string, name string, prodName string) ([]*tag, string, error) {
	// set the tag values that will always be added
	defaultOrganizationTag := resources.GetOrganizationTag()
	clusterID, err := resources.GetClusterID(ctx, c)
	if err != nil {
		msg := "Failed to get cluster id"
		return nil, "", errorUtil.Wrapf(err, msg)
	}
	tags := []*tag{
		{
			key:   defaultOrganizationTag + "clusterID",
			value: clusterID,
		},
		{
			key:   defaultOrganizationTag + "resource-type",
			value: specType,
		},
		{
			key:   defaultOrganizationTag + "resource-name",
			value: name,
		},
		buildManagedTag(),
	}

	if prodName != "" {
		productTag := &tag{
			key:   defaultOrganizationTag + "product-name",
			value: prodName,
		}
		tags = append(tags, productTag)
	}

	infraTags, err := getUserInfraTags(ctx, c)
	if err != nil {
		msg := "Failed to get user infrastructure tags"
		return nil, "", errorUtil.Wrapf(err, msg)
	}
	if infraTags != nil {
		// merge tags into single array, where any duplicate
		// values in infra are overwritten by the default tags
		tags = mergeTags(infraTags, tags)
	}

	return tags, clusterID, nil
}

func getUserInfraTags(ctx context.Context, c client.Client) ([]*tag, error) {
	// get infra CR
	infra, err := resources.GetClusterInfrastructure(ctx, c)
	if err != nil {
		msg := "failed to get cluster infrastructure"
		return nil, errorUtil.Wrapf(err, msg)
	}

	var tags []*tag
	// retrieve all user infrastructure tags
	if infra.Status.PlatformStatus != nil && infra.Status.PlatformStatus.AWS != nil {
		for _, t := range infra.Status.PlatformStatus.AWS.ResourceTags {
			tags = append(tags, &tag{key: t.Key, value: t.Value})
		}
	}
	return tags, nil
}

func buildManagedTag() *tag {
	return &tag{
		key:   tagManagedKey,
		value: tagManagedVal,
	}
}
