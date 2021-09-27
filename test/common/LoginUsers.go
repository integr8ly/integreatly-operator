package common

import (
	"fmt"
	"github.com/integr8ly/integreatly-operator/test/resources"
	"strconv"
	"time"
)

func TestLoginAllUsers(t TestingTB, ctx *TestingContext) {

	user := "test-user0"
	//DefaultPassword := "Password1"

	//for i := 1; i < 10; i++ {
	//	user = "test-user0" + strconv.FormatInt(int64(i), 10)

	for i := 2900; i < 3001; i++ {
		user = "test-user" + strconv.FormatInt(int64(i), 10)

		httpClient, _ := NewTestingHTTPClient(ctx.KubeConfig)

		time.Sleep(100 * time.Millisecond)

		masterURL := "console-openshift-console.apps.briang-byoc.58p6.s1.devshift.org"
		// get rhmi developer user tokens
		//if err := resources.DoAuthOpenshiftUser(fmt.Sprintf("%s/auth/login", masterURL), user, DefaultPassword, ctx.HttpClient, "devsandbox", t); err != nil {
		if err := resources.DoAuthOpenshiftUser( fmt.Sprintf("%s/auth/login", masterURL), user, DefaultPassword, httpClient, "devsandbox", t); err != nil {
			t.Fatalf("error occured trying to get token : %v", err)
		}

		t.Log("here !!!! ", masterURL)

	}
}