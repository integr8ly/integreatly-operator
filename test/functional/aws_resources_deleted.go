package functional

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/integr8ly/integreatly-operator/test/common"
)

func TestAWSResourcesDeleted(t common.TestingTB, ctx *common.TestingContext) {
	goCtx := context.TODO()

	session, err := CreateAWSSession(goCtx, ctx.Client)
	if err != nil {
		t.Fatal("could not create aws session", err)
	}
	ec2Sess := ec2.New(session)

	clusterTag, err := getClusterID(goCtx, ctx.Client)
	if err != nil {
		t.Fatal("could not get cluster id", err)
	}

	vpcs, err := getVpcs(ec2Sess, vpcClusterTagKey, clusterTag)
	if err != nil {
		t.Fatal("could not get vpcs", err)
	}

	if len(vpcs) != 0 {
		t.Errorf("expected no vpcs found, but found the following: %s",
			strings.Join(formatVpcs(vpcs), ", "))
	}
}

func formatVpcs(vpcs []*ec2.Vpc) []string {
	result := make([]string, len(vpcs))

	for i, vpc := range vpcs {
		result[i] = vpc.GoString()
	}

	return result
}
