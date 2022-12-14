package functional

import (
	"context"
	"fmt"
	"github.com/integr8ly/integreatly-operator/test/common"
	"golang.org/x/oauth2/google"
	sqladmin "google.golang.org/api/sqladmin/v1beta4"
)

func TestGCPPostgresSQLInstanceExist(t common.TestingTB, testingContext *common.TestingContext) {

	//List of pSql instances available in Google Cloud Project
	ctx := context.Background()
	c, err := google.DefaultClient(ctx, sqladmin.CloudPlatformScope)
	if err != nil {
		t.Fatal(err)
	}

	sqladminService, err := sqladmin.New(c)
	if err != nil {
		t.Fatal(err)
	}

	projectID, err := getDefaultProjectID(ctx, testingContext.Client)
	if err != nil {
		t.Fatal("error get Default Project ID %w", err)
	}
	var sqlInstanceList []string
	req := sqladminService.Instances.List(projectID)
	if err := req.Pages(ctx, func(page *sqladmin.InstancesListResponse) error {
		for _, databaseInstance := range page.Items {
			// TODO: Change code below to process each `databaseInstance` resource:
			fmt.Printf("databaseInstance: %v\n", databaseInstance)
			sqlInstanceList = append(sqlInstanceList, databaseInstance.Name)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	//Get List of pSql instances from RHOAM CR(s)
	//goContext := context.TODO()
	rhmi, err := common.GetRHMI(testingContext.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}
	pSQLInstanceIdsList, testErrors := GetPostgresSqlInstancesIDsListFromCR(context.TODO(), testingContext.Client, rhmi)
	if len(testErrors) != 0 {
		t.Fatalf("test cro postgres exists failed with the following errors : %s", testErrors)
	}
	for _, psqlId := range pSQLInstanceIdsList {
		if !verifyPostgresSqlInstances(psqlId, sqlInstanceList) {
			t.Fatal("Postgres SQL Instance %s defined in CR, but missing in Google Cloud", psqlId)
		}
	}
}

func verifyPostgresSqlInstances(pSqlInstanceName string, databaseInstanceList []string) bool {
	for _, v := range databaseInstanceList {
		if v == pSqlInstanceName {
			return true
		}
	}
	return false
}
