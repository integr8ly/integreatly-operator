package functional

import "github.com/integr8ly/integreatly-operator/test/common"

var (
	FUNCTIONAL_TESTS_AWS = []common.TestCase{
		{Description: "F01 - Verify AWS rds resources exist and are in expected state", Test: AWSRDSResourcesExistTest},
		{Description: "F03 - Verify AWS elasticache resources exist and are in expected state", Test: AWSElasticacheResourcesExistTest},
		{Description: "A25 - Verify standalone RHMI VPC exists and is configured properly", Test: TestStandaloneVPCExists},
		{Description: "F04 - Verify AWS s3 blob storage resources exist", Test: TestAWSs3BlobStorageResourcesExist},
	}
	FUNCTIONAL_TESTS_GCP = []common.TestCase{
		{Description: "GCP01 - Verify GCP Postgres SQL instances exist", Test: TestGCPPostgresSQLInstanceExist},
		{Description: "GCP02 - Verify GCP Memorystore Redis instances exist", Test: TestGCPMemorystoreRedisInstanceExist},
		{Description: "GCP03 - Verify GCP Network State", Test: TestGCPNetworkState},
		//{Description: "GCP04 - Verify GCP Cloud Storage Blob Storage Resources exist", Test: TestGCPCloudStorageBlobStorageResourcesExist},
	}
	MULTIAZ_TESTS = []common.TestCase{
		{Description: "F09 - Verify correct pod distribution on Multi-AZ cluster", Test: TestMultiAZPodDistribution},
	}
)
