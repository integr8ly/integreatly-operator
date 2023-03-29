package functional

import (
	"context"
	"github.com/integr8ly/cloud-resource-operator/pkg/providers/gcp/gcpiface"
	croResources "github.com/integr8ly/cloud-resource-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/test/common"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
	sqladmin "google.golang.org/api/sqladmin/v1beta4"
)

func TestGCPPostgresSQLInstanceExist(t common.TestingTB, testingContext *common.TestingContext) {

	//List of pSql instances available in Google Cloud Project
	ctx := context.Background()
	serviceAccountJson, err := getGCPCredentials(ctx, testingContext.Client)
	if err != nil {
		t.Fatal("failed to retrieve gcp credentials %v", err)
	}

	sqladminService, err := gcpiface.NewSQLAdminService(ctx, option.WithCredentialsJSON(serviceAccountJson), logrus.NewEntry(logrus.StandardLogger()))
	if err != nil {
		t.Fatalf("error creating sqladmin service %w", err)
	}
	projectID, err := croResources.GetGCPProject(ctx, testingContext.Client)
	if err != nil {
		t.Fatal("error get Default Project ID %w", err)
	}
	rhmi, err := common.GetRHMI(testingContext.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}
	postgresSQLdata, testErrors := GetPostgresInstanceData(context.TODO(), testingContext.Client, rhmi)
	if len(testErrors) != 0 {
		t.Fatalf("test cro postgres exists failed with the following errors : %s", testErrors)
	}
	for clusterInstanceID, postgresVersion := range postgresSQLdata {
		instance, e := sqladminService.GetInstance(ctx, projectID, clusterInstanceID)
		if e != nil {
			t.Fatalf("error getting instance", e)
		}
		if !verifyPostgresSQLInstanceConfig(instance, postgresVersion) {
			t.Fatal("failed as resource is not as expected")
		}
	}
}

func verifyPostgresSQLInstanceConfig(instance *sqladmin.DatabaseInstance, postgresVersion string) bool {
	return labelsContain(instance.Settings.UserLabels, managedLabelKey, managedLabelValue) &&
		instance.Settings.DeletionProtectionEnabled && instance.DatabaseVersion == postgresVersion &&
		instance.Settings.AvailabilityType == "REGIONAL"
}
