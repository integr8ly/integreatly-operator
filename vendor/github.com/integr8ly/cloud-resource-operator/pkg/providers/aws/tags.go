package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/rds"
)

const (
	tagDisplayName = "Name"
)

// generic key-value tag
type tag struct {
	key   string
	value string
}

func ec2TagToGeneric(ec2Tag *ec2.Tag) *tag {
	var genericTag *tag
	genericTag = &tag{key: aws.StringValue(ec2Tag.Key), value: aws.StringValue(ec2Tag.Value)}
	return genericTag
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

func rdsTagstoGeneric(rdsTags []*rds.Tag) []*tag {
	var genericTags []*tag
	for _, rdsTag := range rdsTags {
		genericTags = append(genericTags, &tag{key: aws.StringValue(rdsTag.Key), value: aws.StringValue(rdsTag.Value)})
	}
	return genericTags
}

func tagsContains(tags []*tag, key, value string) bool {
	for _, tag := range tags {
		if tag.key == key && tag.value == value {
			return true
		}
	}
	return false
}
