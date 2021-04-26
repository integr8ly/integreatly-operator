package common

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	threescalev1 "github.com/3scale/3scale-operator/pkg/apis/apps/v1alpha1"
	v12 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	//v12 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	rhmiv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/addon"
	"github.com/integr8ly/integreatly-operator/pkg/products/marin3r"
	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	prometheusRule1           = "api-usage-alert-level1"
	prometheusRule1Desc       = "per minute over 4 hours"
	prometheusRule2           = "api-usage-alert-level2"
	prometheusRule2Desc       = "per minute over 2 hours"
	prometheusRule3           = "api-usage-alert-level3"
	prometheusRule3Desc       = "per minute over 30 minutes"
	higherQuotaname           = "20"
	lowerQuotaname            = "1"
	timeoutWaitingQuotachange = 10
	new3scaleLimits           = "501Mi"
	newKeycloakLimits         = "1501Mi"
	newRatelimitLimits        = "101Mi"
)

func TestQuotaValues(t TestingTB, ctx *TestingContext) {
	quotaConfig, quotaName, err := getQuotaconfig(t, ctx.Client)
	if err != nil {
		t.Fatalf("Error retrieving Quota: %v", err)
	}

	installation, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatal("couldn't get RHMI cr for quota test")
	}
	if installation == nil {
		t.Fatalf("Got invalid rhmi CR: %v", installation)
	}

	// wait if stage no complete
	startTime := time.Now()
	endTime := startTime.Add(time.Minute * time.Duration(timeoutWaitingQuotachange))

	for startTime.Before(endTime) {
		startTime = time.Now()
		if installation.Status.Stage == rhmiv1alpha1.CompleteStage {
			break
		}
	}

	//verify that the TOQuota value is set and that Quota is not set
	//assuming this is run after installation
	if installation.Status.Quota == "" {
		t.Fatal("Quota status not set after installation")
	}
	if installation.Status.ToQuota != "" {
		t.Fatal("toQuota status set after installation")
	}

	if installation.Status.Quota != quotaName {
		t.Fatal(fmt.Sprintf("quota value set as '%s' but doesn't match the expected value: '%s'",
			installation.Status.Quota, quotaName))
	}
	verifyConfiguration(t, ctx.Client, quotaConfig)

	// update the quota to a higher configuration

	t.Logf("Changing Quota to %v million", higherQuotaname)
	installation, err = changeQuota(t, ctx.Client, installation, higherQuotaname)
	if err != nil {
		t.Fatalf("Error changing Quota: %v", err)
	}

	quotaConfig, _, err = getQuotaconfig(t, ctx.Client)
	if err != nil {
		t.Fatalf("Error retrieving Quota config: %v", err)
	}
	verifyConfiguration(t, ctx.Client, quotaConfig)

	// verify that the user can update their configuration manually but it does not get set back
	// get all crs
	threescaleCR := &threescalev1.APIManager{
		ObjectMeta: metav1.ObjectMeta{
			Name:      string(rhmiv1alpha1.Product3Scale),
			Namespace: NamespacePrefix + "3scale",
		},
	}
	key, err := k8sclient.ObjectKeyFromObject(threescaleCR)
	if err != nil {
		t.Fatalf("Error getting APIManager CR key: %v", err)
	}
	new3scaleLimit := resource.MustParse(new3scaleLimits)

	err = ctx.Client.Get(context.TODO(), key, threescaleCR)
	if err != nil && !k8serr.IsNotFound(err) {
		t.Fatalf("Error getting APIManager CR: %v", err)
	}

	// Keycloak
	keycloakCR := &v1alpha1.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      quota.KeycloakName,
			Namespace: NamespacePrefix + "user-sso",
		},
	}
	newKeycloakLimit := resource.MustParse(newKeycloakLimits)

	key, err = k8sclient.ObjectKeyFromObject(keycloakCR)
	if err != nil {
		t.Fatalf("Error getting Keycloak CR key: %v", err)
	}

	err = ctx.Client.Get(context.TODO(), key, keycloakCR)
	if err != nil {
		t.Fatalf("Error getting Keycloak CR: %v", err)
	}

	// Ratelimit
	ratelimitCR := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      quota.RateLimitName,
			Namespace: NamespacePrefix + "marin3r",
		},
	}
	newRatelimitLimit := resource.MustParse(newRatelimitLimits)

	key, err = k8sclient.ObjectKeyFromObject(ratelimitCR)
	if err != nil {
		t.Fatalf("Error getting Ratelimit CR key: %v", err)
	}

	err = ctx.Client.Get(context.TODO(), key, ratelimitCR)
	if err != nil {
		if !k8serr.IsNotFound(err) {
			t.Fatalf("Error getting Ratelimit CR: %v", err)
		}
	}

	// change values in crs to be greater
	t.Log("Increasing limits in products CRs")
	result, err := controllerutil.CreateOrUpdate(context.TODO(), ctx.Client, threescaleCR, func() error {
		threescaleCR.Spec.Backend.ListenerSpec.Resources.Limits[v1.ResourceMemory] = resource.MustParse(new3scaleLimits)
		return nil
	})
	if err != nil {
		t.Fatalf("Error updating APIManager CR: %v with results of: %v", err, result)
	}

	result, err = controllerutil.CreateOrUpdate(context.TODO(), ctx.Client, keycloakCR, func() error {
		keycloakCR.Spec.KeycloakDeploymentSpec.Resources.Limits[v1.ResourceMemory] = resource.MustParse(newKeycloakLimits)
		return nil
	})
	if err != nil {
		t.Fatalf("Error updating Keycloak CR: %v with results of: %v", err, result)
	}

	result, err = controllerutil.CreateOrUpdate(context.TODO(), ctx.Client, ratelimitCR, func() error {
		ratelimitCR.Spec.Template.Spec.Containers[0].Resources.Limits[v1.ResourceMemory] = resource.MustParse(newRatelimitLimits)
		return nil
	})
	if err != nil {
		t.Fatalf("Error updating Ratelimit CR: %v with results of: %v", err, result)
	}

	// wait for 5 minutes and verify that pods have correct values
	startTime = time.Now()
	endTime = startTime.Add(time.Minute * time.Duration(timeoutWaitingQuotachange))

	threescalePods := &v1.PodList{}
	selector, _ := labels.Parse("deploymentConfig=backend-listener")
	threescaleListOpts := []k8sclient.ListOption{
		k8sclient.InNamespace(NamespacePrefix + "3scale"),
		k8sclient.MatchingLabelsSelector{
			Selector: selector,
		},
	}
	keycloakPods := &v1.PodList{}
	selector, _ = labels.Parse("component=keycloak")
	keycloakListOpts := []k8sclient.ListOption{
		k8sclient.InNamespace(NamespacePrefix + "user-sso"),
		k8sclient.MatchingLabelsSelector{
			Selector: selector,
		},
	}
	ratelimitPods := &v1.PodList{}
	selector, _ = labels.Parse("app=ratelimit")
	ratelimitListOpts := []k8sclient.ListOption{
		k8sclient.InNamespace(NamespacePrefix + "marin3r"),
		k8sclient.MatchingLabelsSelector{
			Selector: selector,
		},
	}
	t.Log("Waiting for pods to get updated")
	for startTime.Before(endTime) {
		startTime = time.Now()

		err = ctx.Client.List(context.TODO(), threescalePods, threescaleListOpts...)
		if err != nil {
			t.Fatalf("failed to get backend listener pods for 3scale: %v", err)
		}

		err = ctx.Client.List(context.TODO(), keycloakPods, keycloakListOpts...)
		if err != nil {
			t.Fatalf("failed to get pods for Keycloak: %v", err)
		}

		err = ctx.Client.List(context.TODO(), ratelimitPods, ratelimitListOpts...)
		if err != nil {
			t.Fatalf("failed to get pods for Ratelimit: %v", err)
		}

		// break before the timeout if we are happy
		if podMatchesConfig(threescalePods, new3scaleLimit) &&
			podMatchesConfig(keycloakPods, newKeycloakLimit) &&
			podMatchesConfig(ratelimitPods, newRatelimitLimit) {
			break
		}
		if endTime.Before(startTime) {
			t.Log("Timeout waiting for pods to get updated")
		}
	}

	if !podMatchesConfig(threescalePods, new3scaleLimit) {
		t.Fatalf("3scale backend listener does not have expected memory limits. Expected: %v Got: %v", new3scaleLimit.String(), threescalePods.Items[0].Spec.Containers[0].Resources.Limits.Memory())
	}
	if !podMatchesConfig(keycloakPods, newKeycloakLimit) {
		t.Fatalf("Keycloak pod does not have expected memory limits. Expected: %v Got: %v", newKeycloakLimit.String(), keycloakPods.Items[0].Spec.Containers[0].Resources.Limits.Memory())
	}
	if !podMatchesConfig(ratelimitPods, newRatelimitLimit) {
		t.Fatalf("ratelimit pod does not have expected memory limits. Expected: %v Got: %v", newRatelimitLimit.String(), ratelimitPods.Items[0].Spec.Containers[0].Resources.Limits.Memory())
	}

	t.Logf("Changing Quota to %v million", lowerQuotaname)
	// update to a lower quota
	installation, err = changeQuota(t, ctx.Client, installation, lowerQuotaname)
	if err != nil {
		t.Fatalf("Error changing Quota: %v", err)
	}

	quotaConfig, _, err = getQuotaconfig(t, ctx.Client)
	if err != nil {
		t.Fatalf("Error retrieving Quota config: %v", err)
	}
	verifyConfiguration(t, ctx.Client, quotaConfig)

	t.Log("Yest A34 succeeded")
}

func getConfigMap(_ TestingTB, c k8sclient.Client, name, namespace string) (*v1.ConfigMap, error) {
	configMap := &v1.ConfigMap{}
	if err := c.Get(context.TODO(), k8sclient.ObjectKey{Name: name, Namespace: namespace}, configMap); err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to get '%s' config map in the '%s' namespace", name, namespace))
	}

	return configMap, nil
}

func verifyConfiguration(t TestingTB, c k8sclient.Client, quotaConfig *quota.Quota) {
	// get it from the marin3r namespace
	config, err := getConfigMap(t, c, marin3r.RateLimitingConfigMapName, Marin3rProductNamespace)
	if err != nil {
		t.Fatal(err)
	}

	ratelimit, err := marin3r.GetRateLimitFromConfig(config)
	if err != nil {
		t.Fatal(err)
	}

	configRateLimitRequestPerUnit := quotaConfig.GetRateLimitConfig().RequestsPerUnit

	configRateLimitUnit := quotaConfig.GetRateLimitConfig().Unit

	if ratelimit.RequestsPerUnit != configRateLimitRequestPerUnit {
		t.Fatal(fmt.Sprintf("rate limit requests per unit '%v' does not match the quota config requests per unit '%v'",
			ratelimit.RequestsPerUnit, configRateLimitRequestPerUnit))
	}

	if ratelimit.Unit != configRateLimitUnit {
		t.Fatal(fmt.Sprintf("rate limit unit value '%s' does not match the quota config unit value '%s'",
			ratelimit.Unit, configRateLimitUnit))
	}

	// verify that promethues rules for alerting get update with rate limiting configuration
	prometheusRuleList := &v12.PrometheusRuleList{}
	if err := c.List(context.TODO(), prometheusRuleList, &k8sclient.ListOptions{
		Namespace: Marin3rProductNamespace,
	}); err != nil {
		t.Fatal(fmt.Sprintf("unable to list prometheus rules in namespace '%s'", Marin3rProductNamespace))
	}

	for _, prometheusRule := range prometheusRuleList.Items {
		expr := prometheusRule.Spec.Groups[0].Rules[0].Expr.StrVal
		rateLimitCheck := strconv.Itoa(int(configRateLimitRequestPerUnit * 60 * 4))
		if strings.Contains(prometheusRule.Name, prometheusRule1) != strings.Contains(expr, rateLimitCheck) {
			t.Fatalf(prometheusRateLimitError(rateLimitCheck, prometheusRule1, prometheusRule1Desc, ratelimit.RequestsPerUnit))
		}
		rateLimitCheck = strconv.Itoa(int(configRateLimitRequestPerUnit * 60 * 2))
		if strings.Contains(prometheusRule.Name, prometheusRule2) != strings.Contains(expr, rateLimitCheck) {
			t.Fatalf(prometheusRateLimitError(rateLimitCheck, prometheusRule2, prometheusRule2Desc, ratelimit.RequestsPerUnit))
		}
		rateLimitCheck = strconv.Itoa(int(configRateLimitRequestPerUnit * 30))
		if strings.Contains(prometheusRule.Name, prometheusRule3) != strings.Contains(expr, rateLimitCheck) {
			t.Fatalf(prometheusRateLimitError(rateLimitCheck, prometheusRule3, prometheusRule3Desc, ratelimit.RequestsPerUnit))
		}
	}

	// verify ratelimit replicas and resource configuration is as expected
	configReplicas := quotaConfig.GetProduct(rhmiv1alpha1.ProductMarin3r).GetReplicas(quota.RateLimitName)
	resourceConfig, ok := quotaConfig.GetProduct(rhmiv1alpha1.ProductMarin3r).GetResourceConfig(quota.RateLimitName)
	if !ok {
		t.Fatal("Error obtaining rateLimit resource config")
	}

	ratelimitDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: quota.RateLimitName,
		},
	}
	err = c.Get(context.TODO(), k8sclient.ObjectKey{Name: ratelimitDeployment.Name, Namespace: NamespacePrefix + "marin3r"}, ratelimitDeployment)
	if err != nil {
		t.Fatalf("Couldn't get RateLimit deployment config %v", err)
	}

	crReplicas := *ratelimitDeployment.Spec.Replicas
	crResources := ratelimitDeployment.Spec.Template.Spec.Containers[0].Resources
	checkResources(t, ratelimitDeployment.Name, configReplicas, crReplicas, resourceConfig, crResources)

	// verify rhusersso replicas and resource configuration is as expected
	configReplicas = quotaConfig.GetProduct(quota.KeycloakName).GetReplicas(quota.KeycloakName)
	resourceConfig, ok = quotaConfig.GetProduct(quota.KeycloakName).GetResourceConfig(quota.KeycloakName)
	if !ok {
		t.Fatal("Error obtaining userrhsso resource config")
	}

	// Validate CPU value requested by SSO
	keycloak := &v1alpha1.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name: string(rhmiv1alpha1.ProductRHSSOUser),
		},
	}
	err = c.Get(context.TODO(), k8sclient.ObjectKey{Name: keycloak.Name, Namespace: NamespacePrefix + "user-sso"}, keycloak)
	if err != nil {
		t.Fatalf("Couldn't get Keycloak CR: %v", err)
	}

	crReplicas = int32(keycloak.Spec.Instances)
	crResources = keycloak.Spec.KeycloakDeploymentSpec.Resources
	checkResources(t, keycloak.Name, configReplicas, crReplicas, resourceConfig, crResources)
}

func prometheusRateLimitError(rateLimitCheck, rule, ruseDesc string, requestsPerUnit uint32) string {
	return fmt.Sprintf("the expected value '%v' which is a calculation of ratelimit %v %v is not contained in the prometheus rule expression for rule '%s'", rateLimitCheck, ruseDesc, requestsPerUnit, rule)
}

func checkResources(t TestingTB, productName string, configReplicas, crReplicas int32, resourceConfig, crResources v1.ResourceRequirements) {
	if configReplicas != crReplicas {
		t.Fatalf(fmt.Sprintf("Failed verifying %v replicas: expected %v but got %v ", productName, configReplicas, crReplicas))
	}
	if resourceConfig.Requests.Cpu().Cmp(*crResources.Requests.Cpu()) != 0 {
		t.Fatalf(fmt.Sprintf("Failed verifying %v requested cpu: expected %v but got %v ", productName, resourceConfig.Requests.Cpu(), crResources.Requests.Cpu()))
	}
	if resourceConfig.Requests.Memory().Cmp(*crResources.Requests.Memory()) != 0 {
		t.Fatalf(fmt.Sprintf("Failed verifying %v requested memory: expected %v but got %v ", productName, resourceConfig.Requests.Memory(), resourceConfig.Requests.Memory()))
	}
	if resourceConfig.Limits.Cpu().Cmp(*crResources.Limits.Cpu()) != 0 {
		t.Fatalf(fmt.Sprintf("Failed verifying %v cpu limits: expected %v but got %v ", productName, resourceConfig.Limits.Cpu(), crResources.Limits.Cpu()))
	}
	if resourceConfig.Limits.Memory().Cmp(*crResources.Limits.Memory()) != 0 {
		t.Fatalf(fmt.Sprintf("Failed verifying %v limits: expected %v but got %v ", productName, resourceConfig.Limits.Memory(), resourceConfig.Limits.Memory()))
	}
}

func getQuotaconfig(t TestingTB, c k8sclient.Client) (*quota.Quota, string, error) {
	// verify the config map is in place and can be parsed
	quotaConfigMap, err := getConfigMap(t, c, quota.ConfigMapName, RHMIOperatorNamespace)
	if err != nil {
		t.Fatal(err)
		return nil, "", err
	}

	quotaName, found, err := addon.GetStringParameterByInstallType(context.TODO(), c, rhmiv1alpha1.InstallationTypeManagedApi, RHMIOperatorNamespace, addon.QuotaParamName)
	if !found {
		t.Fatal(fmt.Sprintf("failed to quota parameter '%s' from the parameter secret", addon.QuotaParamName), err)
		return nil, "", err
	}

	quotaConfig := &quota.Quota{}
	err = quota.GetQuota(quotaName, quotaConfigMap, quotaConfig, false)
	if err != nil {
		t.Fatal("failed to get quota config map, skipping test for now until fully implemented", err)
		return nil, "", err
	}

	return quotaConfig, quotaName, nil
}

func changeQuota(t TestingTB, c k8sclient.Client, installation *rhmiv1alpha1.RHMI,
	newQuota string) (*rhmiv1alpha1.RHMI,
	error) {
	newSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "addon-managed-api-service-parameters",
			Namespace: RHMIOperatorNamespace,
		},
	}
	_, err := controllerutil.CreateOrUpdate(context.TODO(), c, newSecret, func() error {
		if newSecret.Data == nil {
			newSecret.Data = make(map[string][]byte, 1)
		}

		newSecret.Data[addon.QuotaParamName] = []byte(newQuota)
		return nil
	})
	if err != nil {
		t.Fatalf("failed updating addon secret with new quota: %v", err)
		return nil, err
	}
	// verifyConfiguration again
	startTime := time.Now()
	endTime := startTime.Add(time.Minute * time.Duration(timeoutWaitingQuotachange))

	t.Log("Waiting for reconciler to apply Quota")
	// break before the timeout if quota was changed
	for startTime.Before(endTime) {
		startTime = time.Now()
		installation, err = GetRHMI(c, true)
		if err == nil && installation.Status.ToQuota == "" && installation.Status.Quota == newQuota {
			break
		}
		if endTime.Before(startTime) {
			t.Log("Timeout waiting for Quota to be changed")
		}
	}
	return installation, nil
}

func podMatchesConfig(podList *v1.PodList, limit resource.Quantity) bool {
	return podList.Items[0].Spec.Containers[0].Resources.Limits.Memory().Cmp(limit) == 0
}
