package common

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	threescalev1 "github.com/3scale/3scale-operator/apis/apps/v1alpha1"
	"github.com/integr8ly/keycloak-client/apis/keycloak/v1alpha1"
	sdk "github.com/openshift-online/ocm-sdk-go"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	v12 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	rhmiv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
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
	higherQuotaParam          = "50"
	lowerQuotaParam           = "1"
	higherQuotaName           = "5 Million"
	higherQuotaParamMT        = "10"
	higherQuotaMTName         = "1 Million"
	lowerQuotaName            = "100K"
	timeoutWaitingQuotachange = 10
	new3scaleLimits           = "501Mi"
	newKeycloakLimits         = "1501Mi"
	newRatelimitLimits        = "101Mi"
	new3scaleLimitsMT         = "1400Mi"
)

func TestQuotaValues(t TestingTB, ctx *TestingContext) {
	quotaConfig, err := getQuotaConfig(t, ctx.Client)
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
		t.Fatal("Expected '.status.quota' to contain a quota name after installation, but got an empty string")
	}
	if installation.Status.ToQuota != "" {
		t.Fatalf("Expected .status.toQuota' to be set after installation to empty value, but got %s", installation.Status.ToQuota)
	}

	if installation.Status.Quota != quotaConfig.GetName() {
		t.Fatalf("Expected quota name '%s' to match the expected value from the quota config map, but got: '%s'",
			installation.Status.Quota, quotaConfig.GetName())
	}
	verifyConfiguration(t, ctx.Client, quotaConfig, installation)

	initialQuotaName := installation.Status.Quota
	initialQuotaValue, found, err := addon.GetStringParameter(context.TODO(), ctx.Client, RHOAMOperatorNamespace, addon.QuotaParamName)
	if !found {
		t.Fatalf("failed to quota parameter '%s' from the parameter secret %v", addon.QuotaParamName, err)
	}
	t.Logf("Initial quota name: %s, value: %s", initialQuotaName, initialQuotaValue)

	// update the quota to a higher configuration if the initial quota is lower,
	// otherwise switch to lower quota configuration
	quotaParam := higherQuotaParam
	quotaName := higherQuotaName

	if rhmiv1alpha1.IsRHOAMMultitenant(rhmiv1alpha1.InstallationType(installation.Spec.Type)) {
		quotaParam = higherQuotaParamMT
		quotaName = higherQuotaMTName
	}

	if initialQuotaValue < quotaParam {
		t.Logf("Changing Quota to %v", quotaName)
		err = changeQuota(t, ctx.Client, quotaParam, quotaName)
	} else {
		t.Logf("Changing Quota to %v", lowerQuotaName)
		err = changeQuota(t, ctx.Client, lowerQuotaParam, lowerQuotaName)
	}

	if err != nil {
		t.Fatalf("Error changing Quota: %v", err)
	}

	// Defer changing quota to initial value in case the test would fail below
	defer func(t TestingTB, c k8sclient.Client, quotaParam, quotaName string) {
		if err := changeQuota(t, c, quotaParam, quotaName); err != nil {
			t.Log(err)
		}
	}(t, ctx.Client, initialQuotaValue, initialQuotaName)

	quotaConfig, err = getQuotaConfig(t, ctx.Client)
	if err != nil {
		t.Fatalf("Error retrieving Quota config: %v", err)
	}
	verifyConfiguration(t, ctx.Client, quotaConfig, installation)

	// verify that the user can update their configuration manually but it does not get set back
	// get all crs
	threescaleCR := &threescalev1.APIManager{
		ObjectMeta: metav1.ObjectMeta{
			Name:      string(rhmiv1alpha1.Product3Scale),
			Namespace: ThreeScaleProductNamespace,
		},
	}
	key := k8sclient.ObjectKeyFromObject(threescaleCR)

	new3scaleLimit := resource.MustParse(new3scaleLimits)
	if rhmiv1alpha1.IsRHOAMMultitenant(rhmiv1alpha1.InstallationType(installation.Spec.Type)) {
		new3scaleLimit = resource.MustParse(new3scaleLimitsMT)
	}

	err = ctx.Client.Get(context.TODO(), key, threescaleCR)
	if err != nil && !k8serr.IsNotFound(err) {
		t.Fatalf("Error getting APIManager CR: %v", err)
	}

	var newKeycloakLimit resource.Quantity
	var keycloakCR *v1alpha1.Keycloak
	if rhmiv1alpha1.IsRHOAMSingletenant(rhmiv1alpha1.InstallationType(installation.Spec.Type)) {
		// Keycloak
		keycloakCR = &v1alpha1.Keycloak{
			ObjectMeta: metav1.ObjectMeta{
				Name:      quota.KeycloakName,
				Namespace: RHSSOUserProductNamespace,
			},
		}
		newKeycloakLimit = resource.MustParse(newKeycloakLimits)

		key = k8sclient.ObjectKeyFromObject(keycloakCR)

		err = ctx.Client.Get(context.TODO(), key, keycloakCR)
		if err != nil {
			t.Fatalf("Error getting Keycloak CR: %v", err)
		}
	}

	// Ratelimit
	ratelimitCR := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      quota.RateLimitName,
			Namespace: Marin3rProductNamespace,
		},
	}
	newRatelimitLimit := resource.MustParse(newRatelimitLimits)

	key = k8sclient.ObjectKeyFromObject(ratelimitCR)

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

	if rhmiv1alpha1.IsRHOAMSingletenant(rhmiv1alpha1.InstallationType(installation.Spec.Type)) {
		result, err = controllerutil.CreateOrUpdate(context.TODO(), ctx.Client, keycloakCR, func() error {
			keycloakCR.Spec.KeycloakDeploymentSpec.Resources.Limits[v1.ResourceMemory] = resource.MustParse(newKeycloakLimits)
			return nil
		})
		if err != nil {
			t.Fatalf("Error updating Keycloak CR: %v with results of: %v", err, result)
		}
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
	selector, err := labels.Parse("deployment=backend-listener")
	if err != nil {
		t.Fatal(err)
	}

	threescaleListOpts := []k8sclient.ListOption{
		k8sclient.InNamespace(ThreeScaleProductNamespace),
		k8sclient.MatchingLabelsSelector{
			Selector: selector,
		},
	}
	keycloakPods := &v1.PodList{}
	selector, err = labels.Parse("component=keycloak")
	if err != nil {
		t.Fatal(err)
	}

	keycloakListOpts := []k8sclient.ListOption{
		k8sclient.InNamespace(RHSSOUserProductNamespace),
		k8sclient.MatchingLabelsSelector{
			Selector: selector,
		},
	}
	ratelimitPods := &v1.PodList{}
	selector, err = labels.Parse("app=ratelimit")
	if err != nil {
		t.Fatal(err)
	}

	ratelimitListOpts := []k8sclient.ListOption{
		k8sclient.InNamespace(Marin3rProductNamespace),
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

		if rhmiv1alpha1.IsRHOAMSingletenant(rhmiv1alpha1.InstallationType(installation.Spec.Type)) {
			err = ctx.Client.List(context.TODO(), keycloakPods, keycloakListOpts...)
			if err != nil {
				t.Fatalf("failed to get pods for Keycloak: %v", err)
			}
		}
		err = ctx.Client.List(context.TODO(), ratelimitPods, ratelimitListOpts...)
		if err != nil {
			t.Fatalf("failed to get pods for Ratelimit: %v", err)
		}

		// break before the timeout if we are happy
		if podMatchesConfig(threescalePods, new3scaleLimit) &&
			(podMatchesConfig(keycloakPods, newKeycloakLimit) || rhmiv1alpha1.IsRHOAMMultitenant(rhmiv1alpha1.InstallationType(installation.Spec.Type))) &&
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
	if !podMatchesConfig(keycloakPods, newKeycloakLimit) && rhmiv1alpha1.IsRHOAMSingletenant(rhmiv1alpha1.InstallationType(installation.Spec.Type)) {
		t.Fatalf("Keycloak pod does not have expected memory limits. Expected: %v Got: %v", newKeycloakLimit.String(), keycloakPods.Items[0].Spec.Containers[0].Resources.Limits.Memory())
	}
	if !podMatchesConfig(ratelimitPods, newRatelimitLimit) {
		t.Fatalf("ratelimit pod does not have expected memory limits. Expected: %v Got: %v", newRatelimitLimit.String(), ratelimitPods.Items[0].Spec.Containers[0].Resources.Limits.Memory())
	}

	// update to a initial quota
	t.Logf("Changing Quota to %v", initialQuotaName)
	err = changeQuota(t, ctx.Client, initialQuotaValue, initialQuotaName)
	if err != nil {
		t.Fatalf("Error changing Quota: %v", err)
	}

	quotaConfig, err = getQuotaConfig(t, ctx.Client)
	if err != nil {
		t.Fatalf("Error retrieving Quota config: %v", err)
	}
	verifyConfiguration(t, ctx.Client, quotaConfig, installation)
}

func getConfigMap(_ TestingTB, c k8sclient.Client, name, namespace string) (*v1.ConfigMap, error) {
	configMap := &v1.ConfigMap{}
	if err := c.Get(context.TODO(), k8sclient.ObjectKey{Name: name, Namespace: namespace}, configMap); err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to get '%s' config map in the '%s' namespace", name, namespace))
	}

	return configMap, nil
}

func verifyConfiguration(t TestingTB, c k8sclient.Client, quotaConfig *quota.Quota, installation *rhmiv1alpha1.RHMI) {
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

	if ratelimit.MaxValue != configRateLimitRequestPerUnit {
		t.Fatal(fmt.Sprintf("rate limit requests per unit '%v' does not match the quota config requests per unit '%v'",
			ratelimit.MaxValue, configRateLimitRequestPerUnit))
	}

	rateLimitUnit, err := marin3r.GetSecondsInUnit(ratelimit.Seconds)

	if err != nil {
		t.Fatal(err)
	}

	if rateLimitUnit != configRateLimitUnit {
		t.Fatal(fmt.Sprintf("rate limit unit value '%s' does not match the quota config unit value '%s'",
			rateLimitUnit, configRateLimitUnit))
	}

	// verify that promethues rules for alerting get update with rate limiting configuration
	prometheusRuleList := &v12.PrometheusRuleList{}
	if err := c.List(context.TODO(), prometheusRuleList, &k8sclient.ListOptions{
		Namespace: ObservabilityProductNamespace,
	}); err != nil {
		t.Fatal(fmt.Sprintf("unable to list prometheus rules in namespace '%s'", ObservabilityProductNamespace))
	}

	for _, prometheusRule := range prometheusRuleList.Items {
		expr := prometheusRule.Spec.Groups[0].Rules[0].Expr.StrVal
		rateLimitCheck := strconv.Itoa(int(configRateLimitRequestPerUnit * 60 * 4))
		if strings.Contains(prometheusRule.Name, prometheusRule1) != strings.Contains(expr, rateLimitCheck) {
			t.Fatalf(prometheusRateLimitError(rateLimitCheck, prometheusRule1, prometheusRule1Desc, ratelimit.MaxValue))
		}
		rateLimitCheck = strconv.Itoa(int(configRateLimitRequestPerUnit * 60 * 2))
		if strings.Contains(prometheusRule.Name, prometheusRule2) != strings.Contains(expr, rateLimitCheck) {
			t.Fatalf(prometheusRateLimitError(rateLimitCheck, prometheusRule2, prometheusRule2Desc, ratelimit.MaxValue))
		}
		rateLimitCheck = strconv.Itoa(int(configRateLimitRequestPerUnit * 30))
		if strings.Contains(prometheusRule.Name, prometheusRule3) != strings.Contains(expr, rateLimitCheck) {
			t.Fatalf(prometheusRateLimitError(rateLimitCheck, prometheusRule3, prometheusRule3Desc, ratelimit.MaxValue))
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
	err = c.Get(context.TODO(), k8sclient.ObjectKey{Name: ratelimitDeployment.Name, Namespace: Marin3rProductNamespace}, ratelimitDeployment)
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

	if rhmiv1alpha1.IsRHOAMSingletenant(rhmiv1alpha1.InstallationType(installation.Spec.Type)) {
		// Validate CPU value requested by SSO
		keycloak := &v1alpha1.Keycloak{
			ObjectMeta: metav1.ObjectMeta{
				Name: string(rhmiv1alpha1.ProductRHSSOUser),
			},
		}
		err = c.Get(context.TODO(), k8sclient.ObjectKey{Name: keycloak.Name, Namespace: RHSSOUserProductNamespace}, keycloak)
		if err != nil {
			t.Fatalf("Couldn't get Keycloak CR: %v", err)
		}

		crReplicas = int32(keycloak.Spec.Instances)
		crResources = keycloak.Spec.KeycloakDeploymentSpec.Resources
		checkResources(t, keycloak.Name, configReplicas, crReplicas, resourceConfig, crResources)
	}
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
		t.Fatalf(fmt.Sprintf("Failed verifying %v requested memory: expected %v but got %v ", productName, resourceConfig.Requests.Memory(), crResources.Requests.Memory()))
	}
	if resourceConfig.Limits.Cpu().Cmp(*crResources.Limits.Cpu()) != 0 {
		t.Fatalf(fmt.Sprintf("Failed verifying %v cpu limits: expected %v but got %v ", productName, resourceConfig.Limits.Cpu(), crResources.Limits.Cpu()))
	}
	if resourceConfig.Limits.Memory().Cmp(*crResources.Limits.Memory()) != 0 {
		t.Fatalf(fmt.Sprintf("Failed verifying %v limits: expected %v but got %v ", productName, resourceConfig.Limits.Memory(), crResources.Limits.Memory()))
	}
}

func getQuotaConfig(t TestingTB, c k8sclient.Client) (*quota.Quota, error) {
	// verify the config map is in place and can be parsed
	quotaConfigMap, err := getConfigMap(t, c, quota.ConfigMapName, RHOAMOperatorNamespace)
	if err != nil {
		t.Fatal(err)
		return nil, err
	}

	quotaParam, found, err := addon.GetStringParameter(context.TODO(), c, RHOAMOperatorNamespace, addon.QuotaParamName)
	if !found {
		t.Fatal(fmt.Sprintf("failed to quota parameter '%s' from the parameter secret", addon.QuotaParamName), err)
		return nil, err
	}

	quotaConfig := &quota.Quota{}
	err = quota.GetQuota(context.TODO(), c, quotaParam, quotaConfigMap, quotaConfig)
	if err != nil {
		t.Fatal("failed GetQuota", err)
		return nil, err
	}

	return quotaConfig, nil
}

func changeQuota(t TestingTB, c k8sclient.Client, quotaParam, quotaName string) error {
	installation, err := GetRHMI(c, true)
	if err != nil {
		t.Fatal("Couldn't get RHMI cr for quota test")
	}
	if installation == nil {
		t.Fatalf("Got invalid rhmi CR: %v", installation)
	}

	if installation.Status.Quota == quotaName {
		t.Logf("changeQuota(): Won't apply a new quota value. Quota %s already configured", quotaName)
		return nil
	}

	hiveManaged, err := addon.OperatorIsHiveManaged(context.TODO(), c, installation)
	if err != nil {
		t.Fatalf("unable to determine if install is hive managed: %v", err)
	}

	if hiveManaged {
		t.Log("quota is hive managed, updating via ocm")
		if err := changeQuotaViaOCM(t, quotaParam); err != nil {
			return err
		}
	} else {
		newSecret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "addon-managed-api-service-parameters",
				Namespace: RHOAMOperatorNamespace,
			},
		}
		_, err = controllerutil.CreateOrUpdate(context.TODO(), c, newSecret, func() error {
			if newSecret.Data == nil {
				newSecret.Data = make(map[string][]byte, 1)
			}

			newSecret.Data[addon.QuotaParamName] = []byte(quotaParam)
			return nil
		})
		if err != nil {
			t.Fatalf("failed updating addon secret with new quota: %v", err)
			return err
		}
	}

	// verifyConfiguration again
	startTime := time.Now()
	endTime := startTime.Add(time.Minute * time.Duration(timeoutWaitingQuotachange))

	t.Log("Waiting for reconciler to apply Quota")
	// break before the timeout if quota was changed
	for startTime.Before(endTime) {
		startTime = time.Now()
		installation, err = GetRHMI(c, true)
		if err == nil && installation.Status.ToQuota == "" && installation.Status.Quota == quotaName {
			break
		}
		if endTime.Before(startTime) {
			t.Log("Timeout waiting for Quota to be changed")
		}
	}
	return nil
}

func changeQuotaViaOCM(t TestingTB, quotaParam string) error {
	token := os.Getenv("OCM_TOKEN")
	if token == "" {
		return fmt.Errorf("OCM_TOKEN must be provided to update quota addon param")
	}

	clusterId := os.Getenv("CLUSTER_ID")
	if clusterId == "" {
		return fmt.Errorf("CLUSTER_ID must be provided to update quota addon param")
	}

	// Create the connection, and remember to close it:
	connection, err := sdk.NewConnectionBuilder().
		URL("https://api.stage.openshift.com").
		Tokens(token).
		Build()
	if err != nil {
		return fmt.Errorf("can't build connection: %v\n", err)
	}
	defer func(connection *sdk.Connection) {
		if err := connection.Close(); err != nil {
			t.Fatal(err)
		}
	}(connection)

	// Get the client for the resource that manages the collection of clusters:
	collection := connection.ClustersMgmt().V1().Clusters()

	addonInstall, err := cmv1.NewAddOnInstallation().
		Addon(cmv1.NewAddOn()).
		Parameters(
			cmv1.NewAddOnInstallationParameterList().
				Items(
					cmv1.NewAddOnInstallationParameter().
						ID("addon-managed-api-service").
						Value(quotaParam),
				),
		).
		Build()
	if err != nil {
		t.Fatalf("Can't build addonInstallation: %v\n", err)
	}

	// Send a request to update the addon parameter for the cluster:
	_, err = collection.Cluster(clusterId).
		Addons().
		Addoninstallation("managed-api-service").
		Update().
		Body(addonInstall).
		Send()
	if err != nil {
		return fmt.Errorf("can't update addon quota parameter: %v\n", err)
	}

	return nil
}

func podMatchesConfig(podList *v1.PodList, limit resource.Quantity) bool {
	if len(podList.Items) == 0 {
		return false
	}
	return podList.Items[0].Spec.Containers[0].Resources.Limits.Memory().Cmp(limit) == 0
}
