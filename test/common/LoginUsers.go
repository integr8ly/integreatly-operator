package common

import (
	"fmt"
	"github.com/integr8ly/integreatly-operator/test/resources"
	"strconv"
)

func TestLoginAllUsers(t TestingTB, ctx *TestingContext) {

	user := "test-user0"
	//DefaultPassword := "Password1"

	for i := 10; i < 3000; i++ {
		user = "test-user" + strconv.FormatInt(int64(i), 10)

		httpClient, _ := NewTestingHTTPClient(ctx.KubeConfig)

		masterURL := "console-openshift-console.apps.bg-byoc.lgqh.s1.devshift.org"
		// get rhmi developer user tokens
		//if err := resources.DoAuthOpenshiftUser(fmt.Sprintf("%s/auth/login", masterURL), user, DefaultPassword, ctx.HttpClient, "devsandbox", t); err != nil {
		if err := resources.DoAuthOpenshiftUser( fmt.Sprintf("%s/auth/login", masterURL), user, DefaultPassword, httpClient, "devsandbox", t); err != nil {
			t.Fatalf("error occured trying to get token : %v", err)
		}

		t.Log("here !!!! ", masterURL)

	}
}