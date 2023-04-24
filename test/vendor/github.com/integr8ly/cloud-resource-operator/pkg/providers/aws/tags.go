package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/integr8ly/cloud-resource-operator/pkg/resources"
)

func ec2TagToGeneric(ec2Tag *ec2.Tag) *resources.Tag {
	return &resources.Tag{Key: aws.StringValue(ec2Tag.Key), Value: aws.StringValue(ec2Tag.Value)}
}

func ec2TagListToGenericList(ec2Tags []*ec2.Tag) []*resources.Tag {
	var genericTags []*resources.Tag
	for _, ec2Tag := range ec2Tags {
		genericTags = append(genericTags, &resources.Tag{Key: aws.StringValue(ec2Tag.Key), Value: aws.StringValue(ec2Tag.Value)})
	}
	return genericTags
}

func genericToEc2Tag(tag *resources.Tag) *ec2.Tag {
	return &ec2.Tag{Key: aws.String(tag.Key), Value: aws.String(tag.Value)}
}

func genericToRdsTag(tag *resources.Tag) *rds.Tag {
	return &rds.Tag{Key: aws.String(tag.Key), Value: aws.String(tag.Value)}
}

func genericToS3Tag(tag *resources.Tag) *s3.Tag {
	return &s3.Tag{Key: aws.String(tag.Key), Value: aws.String(tag.Value)}
}

func genericToElasticacheTag(tag *resources.Tag) *elasticache.Tag {
	return &elasticache.Tag{Key: aws.String(tag.Key), Value: aws.String(tag.Value)}
}

func genericToRdsTags(tags []*resources.Tag) []*rds.Tag {
	var rdsTags []*rds.Tag
	for _, tag := range tags {
		rdsTags = append(rdsTags, genericToRdsTag(tag))
	}
	return rdsTags
}

func genericToS3Tags(tags []*resources.Tag) []*s3.Tag {
	var s3Tags []*s3.Tag
	for _, tag := range tags {
		s3Tags = append(s3Tags, genericToS3Tag(tag))
	}
	return s3Tags
}

func genericListToElasticacheTagList(tags []*resources.Tag) []*elasticache.Tag {
	var cacheTags []*elasticache.Tag
	for _, tag := range tags {
		cacheTags = append(cacheTags, genericToElasticacheTag(tag))
	}
	return cacheTags
}

func rdsTagListToGenericList(rdsTags []*rds.Tag) []*resources.Tag {
	var genericTags []*resources.Tag
	for _, rdsTag := range rdsTags {
		genericTags = append(genericTags, &resources.Tag{Key: aws.StringValue(rdsTag.Key), Value: aws.StringValue(rdsTag.Value)})
	}
	return genericTags
}

func genericListToEc2TagList(tags []*resources.Tag) []*ec2.Tag {
	var ec2Tags []*ec2.Tag
	for _, tag := range tags {
		ec2Tags = append(ec2Tags, genericToEc2Tag(tag))
	}
	return ec2Tags
}
