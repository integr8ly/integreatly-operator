package common

import (
	goctx "context"
	"testing"

	"github.com/PuerkitoBio/goquery"
	v1 "github.com/openshift/api/route/v1"
	"gopkg.in/headzoo/surf.v1"
	"k8s.io/apimachinery/pkg/types"
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCreateAddressSpace(t *testing.T, ctx *TestingContext) {

	// console\Url getAMQOnlineConsoleUrl(ctx.Client)

	browser := surf.NewBrowser()

	// https://grafana-route-redhat-rhmi-middleware-monitoring-operator.apps.dbizzarr.m4y1.s1.devshift.org/
	err := browser.Open("https://console-redhat-rhmi-amq-online.apps.dbizzarr.m4y1.s1.devshift.org/oauth/start")
	if err != nil {
		t.Fatalf("%s", err)
	}

	// <noscript> bug workaround
	//
	// https://github.com/PuerkitoBio/goquery/issues/139#issuecomment-517526070
	browser.Find("noscript").Each(func(i int, s *goquery.Selection) {
		s.SetHtml(s.Text())
	})

	browser.Click("a:contains('testing-idp')")

	f, err := browser.Form("#kc-form-login")
	if err != nil {
		t.Fatalf("%s", err)
	}

	err = f.Input("username", "customer-admin01")
	if err != nil {
		t.Fatalf("%s", err)
	}
	err = f.Input("password", "Password1")
	if err != nil {
		t.Fatalf("%s", err)
	}

	err = f.Submit()
	if err != nil {
		t.Fatalf("%s", err)
	}

	t.Log(browser.Url())
	t.Log(browser.ResponseHeaders())
	t.Log(browser.Body())
	t.Log(browser.SiteCookies()[1])
}

func getAMQOnlineConsoleUrl(client dynclient.Client) (string, error) {

	var r v1.Route
	err := client.Get(goctx.TODO(), types.NamespacedName{Namespace: "redhat-rhmi-amq-online", Name: "console"}, &r)
	if err != nil {
		return "", err
	}

	return r.Spec.Host, nil
}
