package common

import (
	"context"
	"fmt"
	v12 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"strconv"
	"strings"

	//v12 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	rhmiv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/addon"
	"github.com/integr8ly/integreatly-operator/pkg/products/marin3r"
	"github.com/integr8ly/integreatly-operator/pkg/resources/sku"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestSKUValues(t TestingTB, ctx *TestingContext) {

	// verify the config map is in place and can be parsed
	skuConfigMap, err := getConfigMap(t, ctx.Client, sku.ConfigMapName, RHMIOperatorNamespace)
	if err != nil {
		t.Fatal(err)
	}

	quotaName, found, err := addon.GetStringParameterByInstallType(context.TODO(), ctx.Client, rhmiv1alpha1.InstallationTypeManagedApi, RHMIOperatorNamespace, addon.QuotaParamName)
	if !found {
		t.Fatal(fmt.Sprintf("failed to sku parameter '%s' from the parameter secret", addon.QuotaParamName), err)
		return
	}


	skuConfig := &sku.SKU{}
	err = sku.GetSKU(quotaName, skuConfigMap, skuConfig,false)
	if err != nil {
		t.Fatal("failed to get sku config map, skipping test for now until fully implemented", err)
	}

	installation, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatal("couldn't get RHMI cr for sku test")
	}

	//verify that the TOSKU value is set and that SKU is not set
	//assuming this is run after installation
	if installation.Status.SKU == "" {
		t.Fatal("SKU status not set after installation")
	}
	if installation.Status.ToSKU != "" {
		t.Fatal("toSKU status set after installation")
	}

	if installation.Status.SKU != quotaName {
		t.Fatal(fmt.Sprintf("sku value set as '%s' but doesn't match the expected value: '%s'", installation.Status.SKU, quotaName))
	}

	verifyConfiguration(t, ctx.Client, skuConfig)

	// TODO update the sku to a higher configuration
	// verifyConfiguration again

	// TODO verify that the user can update their configuration manually but it does not get set back

	// TODO update to a lower sku
	// verifyConfiguration again


}

func getConfigMap(_ TestingTB, c k8sclient.Client, name, namespace string) (*v1.ConfigMap, error) {
	configMap := &v1.ConfigMap{}
	if err := c.Get(context.TODO(), k8sclient.ObjectKey{Name: name, Namespace: namespace}, configMap); err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to get '%s' config map in the '%s' namespace", name, namespace))
	}

	return configMap, nil
}

func verifyConfiguration(t TestingTB, c k8sclient.Client, skuConfig *sku.SKU) {

	// TODO verify that the sku configuration is as expected
	// get it from the marin3r namespace
	config, err := getConfigMap(t, c, marin3r.RateLimitingConfigMapName, Marin3rProductNamespace)
	if err != nil {
		t.Fatal(err)
	}
	ratelimit, err := marin3r.GetRateLimitFromConfig(config)
	if err != nil {
		t.Fatal(err)
	}

	configRateLimitRequestPerUnit := skuConfig.GetRateLimitConfig().RequestsPerUnit
	configRateLimitUnit := skuConfig.GetRateLimitConfig().Unit

	if ratelimit.RequestsPerUnit != configRateLimitRequestPerUnit {
		t.Fatal(fmt.Sprintf("rate limit requests per unit '%v' does not match the sku config requests per unit '%v'", ratelimit.RequestsPerUnit, configRateLimitRequestPerUnit))
	}

	if ratelimit.Unit != configRateLimitUnit {
		t.Fatal(fmt.Sprintf("rate limit unit value '%s' does not match the sku config unit value '%s'", ratelimit.Unit, configRateLimitUnit))
	}


	// TODO verify that promethues rules for alerting get update with rate limiting configuration
	prometheusRuleList := &v12.PrometheusRuleList{}
	if err := c.List(context.TODO(), prometheusRuleList, &k8sclient.ListOptions{
		Namespace: Marin3rProductNamespace,
	}); err != nil {
		t.Fatal(fmt.Sprintf("unable to list prometheus rules in namespace '%s'", Marin3rProductNamespace))
	}


	for _, prometheusRule := range prometheusRuleList.Items {
		if strings.Contains(prometheusRule.Name, "api-usage-alert-level") {
			var rateLimitCheck uint32
			if strings.Contains(prometheusRule.Name , "1") {
				rateLimitCheck = configRateLimitRequestPerUnit * 60 * 4
			} else if strings.Contains(prometheusRule.Name , "2") {
				rateLimitCheck = configRateLimitRequestPerUnit * 60 * 2
			} else if strings.Contains(prometheusRule.Name , "3") {
				rateLimitCheck = configRateLimitRequestPerUnit * 30
			}
			if !strings.Contains(prometheusRule.Spec.Groups[0].Rules[0].Expr.StrVal, strconv.Itoa(int(rateLimitCheck))) {
				t.Fatal(fmt.Sprintf("the expected value '%v' which is a calculation of ratelimit per minute over 4 hours %v*60*4 is not contained in the prometheus rule expression for rule '%s'", rateLimitCheck, ratelimit.RequestsPerUnit, prometheusRule.Name))
			}
		}
	}

	// TODO verify that grafana dashboard(s) has the expected rate limiting configuration

	// TODO verify ratelimit replicas and resource configuration is as expected

	// TODO verify rhusersso replicas and resource configuration is as expected

	// verify 3scale replicas and resource configuration is as expected
	// TODO when 3scale work is merged

}
