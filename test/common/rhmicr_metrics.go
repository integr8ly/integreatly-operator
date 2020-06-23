package common

import (
	goctx "context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestRHMICRMetrics(t *testing.T, ctx *TestingContext) {

	rhmi, err := getRHMI(ctx.Client)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}

	rhmiOperatorPod, err := getRHMIOperatorPod(ctx)
	if err != nil {
		t.Fatalf("error getting rhmi-operator pod: %v", err)
	}

	output, err := execToPod("curl --silent localhost:8383/metrics",
		rhmiOperatorPod.GetName(),
		RHMIOperatorNamespace,
		"rhmi-operator",
		ctx)
	if err != nil {
		t.Fatalf("failed to exec to prometheus pod: %w", err)
	}

	// check if rhmi_status is present
	rhmiStatusMetricPresent := regexp.MustCompile(`rhmi_status{.*}`)
	if !rhmiStatusMetricPresent.MatchString(output) {
		t.Fatalf("rhmi_status metric is not present: %w", err)
	}

	// check if the metric labels matches rhmi CR
	stringRHMIStatus := rhmiStatusMetricPresent.FindString(output)
	labels := parsePrometheusMetricToMap(stringRHMIStatus, "rhmi_status")
	if labels["stage"] != string(rhmi.Status.Stage) {
		t.Fatalf("rhmi_status metric stage does not match current stage: %s != %s", labels["stage"], string(rhmi.Status.Stage))
	}

	// check if rhmi_info is present
	rhmiInfoMetricPresent := regexp.MustCompile(`rhmi_spec{.*}`)
	if !rhmiInfoMetricPresent.MatchString(output) {
		t.Fatalf("rhmi_spec metric is not present: %w", err)
	}

	// check if rhmi_info metric labels matches with rhmi installation CR
	stringRHMIInfo := rhmiInfoMetricPresent.FindString(output)
	rhmiInfoLabels := parsePrometheusMetricToMap(stringRHMIInfo, "rhmi_spec")

	doRHMIInfoLabelsMatch := true
	if rhmiInfoLabels["use_cluster_storage"] != rhmi.Spec.UseClusterStorage ||
		rhmiInfoLabels["master_url"] != rhmi.Spec.MasterURL ||
		rhmiInfoLabels["installation_type"] != rhmi.Spec.Type ||
		rhmiInfoLabels["operator_name"] != rhmi.GetName() ||
		rhmiInfoLabels["namespace"] != rhmi.GetNamespace() ||
		rhmiInfoLabels["namespace_prefix"] != rhmi.Spec.NamespacePrefix ||
		rhmiInfoLabels["operators_in_product_namespace"] != fmt.Sprintf("%t", rhmi.Spec.OperatorsInProductNamespace) ||
		rhmiInfoLabels["routing_subdomain"] != rhmi.Spec.RoutingSubdomain ||
		rhmiInfoLabels["self_signed_certs"] != fmt.Sprintf("%t", rhmi.Spec.SelfSignedCerts) {
		doRHMIInfoLabelsMatch = false
	}
	if !doRHMIInfoLabelsMatch {
		t.Fatalf("rhmi_info metric labels do not match with rhmi CR: %w", err)
	}
}

func getRHMIOperatorPod(ctx *TestingContext) (*corev1.Pod, error) {

	listOptions := []k8sclient.ListOption{
		k8sclient.MatchingLabels(map[string]string{
			"name": "rhmi-operator",
		}),
		k8sclient.InNamespace(RHMIOperatorNamespace),
	}

	rhmiOperatorPod := &corev1.PodList{}

	err := ctx.Client.List(goctx.TODO(), rhmiOperatorPod, listOptions...)
	if err != nil {
		return nil, fmt.Errorf("error listing rhmi-operator pod: %v", err)
	}

	if len(rhmiOperatorPod.Items) < 0 {
		return nil, fmt.Errorf("rhmi-operator pod doesn't exist: %v", err)
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
