package common

import (
	//"github.com/integr8ly/integreatly-operator/test/resources"
	//"fmt"
	"strconv"
)

func TestLoginAllUsers(t TestingTB, ctx *TestingContext) {

	user := "test-user0"
	//DefaultPassword := "Password1"

	for i := 1; i < 10; i++ {
		user = user + strconv.FormatInt(int64(i), 10)


		masterURL := "https://console-openshift-console.apps.bg-byoc.lgqh.s1.devshift.org"
		// get rhmi developer user tokens
		//if err := resources.DoAuthOpenshiftUser(fmt.Sprintf("%s/auth/login", masterURL), user, DefaultPassword, ctx.HttpClient, "devsandbox", t); err != nil {
		//	t.Fatalf("error occured trying to get token : %v", err)
		//}

		t.Log("here !!!! ", masterURL)

	}
}