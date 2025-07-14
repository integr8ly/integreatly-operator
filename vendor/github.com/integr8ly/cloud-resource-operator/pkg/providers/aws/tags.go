package aws

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	elasticachetypes "github.com/aws/aws-sdk-go-v2/service/elasticache/types"
	rdstypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/integr8ly/cloud-resource-operator/pkg/resources"
)

func ec2TagToGeneric(ec2Tag *ec2types.Tag) *resources.Tag {
	return &resources.Tag{Key: aws.ToString(ec2Tag.Key), Value: aws.ToString(ec2Tag.Value)}
}

func ec2TagListToGenericList(ec2Tags []ec2types.Tag) []*resources.Tag {
	var genericTags []*resources.Tag
	for _, ec2Tag := range ec2Tags {
		genericTags = append(genericTags, &resources.Tag{Key: aws.ToString(ec2Tag.Key), Value: aws.ToString(ec2Tag.Value)})
	}
	return genericTags
}

func genericToEc2Tag(tag *resources.Tag) *ec2types.Tag {
	return &ec2types.Tag{Key: aws.String(tag.Key), Value: aws.String(tag.Value)}
}

func genericToRdsTag(tag *resources.Tag) *rdstypes.Tag {
	return &rdstypes.Tag{Key: aws.String(tag.Key), Value: aws.String(tag.Value)}
}

func genericToS3Tag(tag *resources.Tag) types.Tag {
	return types.Tag{Key: aws.String(tag.Key), Value: aws.String(tag.Value)}
}

func genericToElasticacheTag(tag *resources.Tag) *elasticachetypes.Tag {
	return &elasticachetypes.Tag{Key: aws.String(tag.Key), Value: aws.String(tag.Value)}
}

func genericToRdsTags(tags []*resources.Tag) []rdstypes.Tag {
	var rdsTags []rdstypes.Tag
	for _, tag := range tags {
		rdsTags = append(rdsTags, *genericToRdsTag(tag))
	}
	return rdsTags
}

func genericToS3Tags(tags []*resources.Tag) []types.Tag {
	var s3Tags []types.Tag
	for _, tag := range tags {
		s3Tags = append(s3Tags, genericToS3Tag(tag))
	}
	return s3Tags
}

func genericListToElasticacheTagList(tags []*resources.Tag) []elasticachetypes.Tag {
	var cacheTags []elasticachetypes.Tag
	for _, tag := range tags {
		cacheTags = append(cacheTags, *genericToElasticacheTag(tag))
	}
	return cacheTags
}

func rdsTagListToGenericList(rdsTags []rdstypes.Tag) []*resources.Tag {
	var genericTags []*resources.Tag
	for _, rdsTag := range rdsTags {
		genericTags = append(genericTags, &resources.Tag{Key: aws.ToString(rdsTag.Key), Value: aws.ToString(rdsTag.Value)})
	}
	return genericTags
}

func genericListToEc2TagList(tags []*resources.Tag) []ec2types.Tag {
	var ec2Tags []ec2types.Tag
	for _, tag := range tags {
		ec2Tags = append(ec2Tags, *genericToEc2Tag(tag))
	}
	return ec2Tags
}
