package common

import (
	goctx "context"
	"fmt"
	"regexp"
	"strings"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestRHMICRMetrics(t TestingTB, ctx *TestingContext) {

	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}

	rhmiOperatorPod, err := getRHMIOperatorPod(ctx)
	if err != nil {
		t.Fatalf("error getting rhmi-operator pod: %v", err)
	}

	output, err := execToPod("curl --silent localhost:8383/metrics",
		rhmiOperatorPod.Name,
		RHOAMOperatorNamespace,
		"rhmi-operator",
		ctx)
	if err != nil {
		t.Fatalf("failed to exec to prometheus pod: %v", err)
	}

	// check if rhoam_status is present
	rhoamStatusMetricPresent := regexp.MustCompile(`rhoam_status{.*}`)
	if !rhoamStatusMetricPresent.MatchString(output) {
		t.Fatalf("rhoam_status metric is not present. Metrics output:\n%v", output)
	}

	// check if the metric labels matches rhmi CR
	stringRHOAMStatus := rhoamStatusMetricPresent.FindString(output)
	labels := parsePrometheusMetricToMap(stringRHOAMStatus, "rhoam_status")
	if labels["stage"] != string(rhmi.Status.Stage) {
		t.Fatalf("rhoam_status metric stage does not match current stage: %s != %s", labels["stage"], string(rhmi.Status.Stage))
	}

	// check if rhoam_spec is present
	rhoamInfoMetricPresent := regexp.MustCompile(`rhoam_spec{.*}`)
	if !rhoamInfoMetricPresent.MatchString(output) {
		t.Fatalf("rhoam_spec metric is not present. Metrics output:\n%v", output)
	}

	// check if rhmi_info metric labels matches with rhmi installation CR
	stringRHMIInfo := rhoamInfoMetricPresent.FindString(output)
	infoLabels := parsePrometheusMetricToMap(stringRHMIInfo, "rhoam_spec")

	doInfoLabelsMatch := true
	if infoLabels["use_cluster_storage"] != rhmi.Spec.UseClusterStorage ||
		infoLabels["master_url"] != rhmi.Spec.MasterURL ||
		infoLabels["installation_type"] != rhmi.Spec.Type ||
		infoLabels["operator_name"] != rhmi.GetName() ||
		infoLabels["namespace"] != rhmi.GetNamespace() ||
		infoLabels["namespace_prefix"] != rhmi.Spec.NamespacePrefix ||
		infoLabels["operators_in_product_namespace"] != fmt.Sprintf("%t", rhmi.Spec.OperatorsInProductNamespace) ||
		infoLabels["routing_subdomain"] != rhmi.Spec.RoutingSubdomain ||
		infoLabels["self_signed_certs"] != fmt.Sprintf("%t", rhmi.Spec.SelfSignedCerts) {
		doInfoLabelsMatch = false
	}
	if !doInfoLabelsMatch {
		t.Fatalf("rhmi_info metric labels do not match with rhmi CR. Labels:\n%v", infoLabels)
	}
}

func getRHMIOperatorPod(ctx *TestingContext) (*corev1.Pod, error) {
	listOptions := []k8sclient.ListOption{
		k8sclient.MatchingLabels(map[string]string{
			"name": "rhmi-operator",
		}),
		k8sclient.InNamespace(RHOAMOperatorNamespace),
	}

	rhmiOperatorPod := &corev1.PodList{}

	err := ctx.Client.List(goctx.TODO(), rhmiOperatorPod, listOptions...)
	if err != nil {
		return nil, fmt.Errorf("error listing rhmi-operator pod: %v", err)
	}

	if len(rhmiOperatorPod.Items) == 0 {
		return nil, fmt.Errorf("%v pod doesn't exist in namespace: %v (err: %v)", podName, RHOAMOperatorNamespace, err)
	}

	return &rhmiOperatorPod.Items[0], nil
}

func parsePrometheusMetricToMap(metric, metricName string) map[string]string {
	// remove unwanted part of the string
	metric = strings.ReplaceAll(metric, fmt.Sprintf("%s{", metricName), "")
	metric = strings.ReplaceAll(metric, "}", "")

	labelsWithValue := strings.Split(metric, ",")
	parsedStrings := map[string]string{}
	for _, labelAndValue := range labelsWithValue {
		value := strings.Split(labelAndValue, "=")
		parsedStrings[value[0]] = strings.ReplaceAll(value[1], "\"", "")
	}
	return parsedStrings
}

func sanitizeForPrometheusLabel(productName integreatlyv1alpha1.ProductName) string {
	if productName == integreatlyv1alpha1.Product3Scale {
		productName = "threescale"
	}
	return fmt.Sprintf("%s_status", strings.ReplaceAll(string(productName), "-", "_"))
}
