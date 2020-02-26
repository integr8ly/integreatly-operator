package common

import (
	"bytes"
	goctx "context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"

	"k8s.io/apimachinery/pkg/types"

	corev1 "k8s.io/api/core/v1"

	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/remotecommand"
)

var expectedRules = map[string][]string{
	"/redhat-rhmi-middleware-monitoring": {
		"DeadMansSwitch",
		"KubePodBadConfig",
		"KubePodCrashLooping",
		"KubePodImagePullBackOff",
		"KubePodNotReady",
		"KubePodStuckCreating",
		"ClusterSchedulableMemoryLow",
		"ClusterSchedulableCPULow",
		"PVCStorageAvailable",
		"PVCStorageMetricsAvailable",
		"CronJobNotRunInThreshold",
		"CronJobSuspended",
		"CronJobsFailed",
		"JobRunningTimeExceeded",
		"JobRunningTimeExceeded",
		"MiddlewareMonitoringPodCount",
	},
	"/redhat-rhmi-3scale": {
		"ThreeScaleAdminUIBBT",
		"ThreeScaleApicastProductionPod",
		"ThreeScaleApicastStagingPod",
		"ThreeScaleBackendListenerPod",
		"ThreeScaleBackendWorkerPod",
		"ThreeScaleDeveloperUIBBT",
		"ThreeScalePodCount",
		"ThreeScalePodHighCPU",
		"ThreeScalePodHighMemory",
		"ThreeScaleSystemAdminUIBBT",
		"ThreeScaleSystemAppPod",
	},
	"/redhat-rhmi-amq-online": {
		"AMQOnlinePodCount",
		"AMQOnlinePodHighMemory",
		"AMQOnlineConsoleAvailable",
		"AMQOnlineKeycloakAvailable",
		"AMQOnlineOperatorAvailable",
		"CronJobExists_redhat-rhmi-amq-online_enmasse-pv-backup",
	},
	"/redhat-rhmi-fuse": {
		"FuseOnlinePodCount",
		"FuseOnlineSyndesisServerInstanceDown",
		"FuseOnlineSyndesisUIInstanceDown",
		"FuseOnlineDatabaseInstanceDown",
		"FuseOnlinePostgresExporterDown",
		"FuseOnlineRestApiHighEndpointErrorRate",
		"FuseOnlineRestApiHighEndpointLatency",
		"FuseOnlineRestApiHighEndpointErrorRate",
		"FuseOnlineRestApiHighEndpointLatency",
		"IntegrationExchangesHighFailureRate",
	},
	"/redhat-rhmi-codeready": {
		"CodeReadyPodCount",
		"CronJobExists_redhat-rhmi-codeready-workspaces_codeready-pv-backup",
	},
	"/redhat-rhmi-solutionexplorer": {
		"SolutionExplorerPodCount",
	},
	"/redhat-rhmi-ups": {
		"UnifiedPushOperatorDown",
		"UnifiedPushConsoleDown",
		"UnifiedPushDown",
		"UnifiedPushJavaDeadlockedThreads",
		"UnifiedPushJavaGCTimePerMinuteScavenge",
		"UnifiedPushJavaHeapThresholdExceeded",
		"UnifiedPushJavaNonHeapThresholdExceeded",
		"UnifiedPushMessagesFailures",
	},
	"/redhat-rhmi-rhsso": {
		"KeycloakAPIRequestDuration90PercThresholdExceeded",
		"KeycloakAPIRequestDuration99.5PercThresholdExceeded",
		"KeycloakInstanceNotAvailable",
		"KeycloakJavaDeadlockedThreads",
		"KeycloakJavaGCTimePerMinuteMarkSweep",
		"KeycloakJavaGCTimePerMinuteScavenge",
		"KeycloakJavaHeapThresholdExceeded",
		"KeycloakJavaNonHeapThresholdExceeded",
		"KeycloakLoginFailedThresholdExceeded",
	},
	"/redhat-rhmi-user-sso": {
		"KeycloakAPIRequestDuration90PercThresholdExceeded",
		"KeycloakAPIRequestDuration99.5PercThresholdExceeded",
		"KeycloakInstanceNotAvailable",
		"KeycloakJavaDeadlockedThreads",
		"KeycloakJavaGCTimePerMinuteMarkSweep",
		"KeycloakJavaGCTimePerMinuteScavenge",
		"KeycloakJavaHeapThresholdExceeded",
		"KeycloakJavaNonHeapThresholdExceeded",
		"KeycloakLoginFailedThresholdExceeded",
	},
}

var expectedExtRules = map[string][]string{
	"/redhat-rhmi-3scale": {
		"ThreeScalePostgresUnavailable",
		"ThreeScalePostgresConnectivity",
		"BackendRedisUnavailable",
		"BackendRedisConnectivity",
		"SystemRedisUnavailable",
		"SystemRedisConnectivity",
	},
	"/redhat-rhmi-codeready": {
		"CodeReadyPostgresUnavailable",
		"CodeReadyPostgresConnectivity",
	},
	"/redhat-rhmi-ups": {
		"UPSPostgresUnavailable",
		"UPSPostgresConnectivity",
	},
	"/redhat-rhmi-user-sso": {
		"UserSSOPostgresUnavailable",
		"UserSSOPostgresConnectivity",
	},
	"/redhat-rhmi-sso": {
		"SSOPostgresUnavailable",
		"SSOPostgresConnectivity",
	},
	"/redhat-rhmi-amq-online": {
		"AMQOnlineSSOPostgresUnavailable",
		"AMQOnlineSSOPostgresConnectivity",
	},
}

func TestIntegreatlyAlertsExist(t *testing.T, ctx *TestingContext) {
	// get the RHMI custom resource to check what storage type is being used
	rhmi := &v1alpha1.RHMI{}
	ns := fmt.Sprintf("%soperator", namespacePrefix)
	err := ctx.Client.Get(goctx.TODO(), types.NamespacedName{Name: installationName, Namespace: ns}, rhmi)
	if err != nil {
		t.Fatal("error getting RHMI CR:", err)
	}

	// add external database alerts to list of expected rules if
	// cluster storage is not being used
	if rhmi.Spec.UseClusterStorage != "true" {
		for file, rules := range expectedExtRules {
			expectedRules[file] = append(expectedRules[file], rules...)
		}
	}

	// exec into the prometheus pod
	output, err := execToPod("curl localhost:9090/api/v1/rules",
		"prometheus-application-monitoring-0",
		namespacePrefix+"middleware-monitoring-operator",
		"prometheus", ctx)
	if err != nil {
		t.Fatal("failed to exec to pod:", err)
	}

	// get all found rules from the prometheus api
	var promApiCallOutput prometheusAPIResponse
	err = json.Unmarshal([]byte(output), &promApiCallOutput)
	if err != nil {
		t.Fatal("Failed to unmarshal json:", err)
	}
	var rulesResult prometheusv1.RulesResult
	err = json.Unmarshal([]byte(promApiCallOutput.Data), &rulesResult)
	if err != nil {
		t.Fatal("Failed to unmarshal json:", err)
	}

	var diff []string
	actualRules := make(map[string][]string, 0)
	for file, rules := range expectedRules {
		// build a map of found rules for each file name. The difference
		// between this actual map and the expected map will provide any missing rules
		for _, group := range rulesResult.Groups {
			if !strings.Contains(group.File, file) {
				continue
			}
			// create an empty entry in the map
			if _, ok := actualRules[file]; !ok {
				actualRules[file] = []string{}
			}
			// add found rules to the actual rules map
			for _, rule := range group.Rules {
				switch v := rule.(type) {
				case prometheusv1.RecordingRule:
					rule := rule.(prometheusv1.RecordingRule)
					actualRules[file] = append(actualRules[file], rule.Name)
				case prometheusv1.AlertingRule:
					rule := rule.(prometheusv1.AlertingRule)
					actualRules[file] = append(actualRules[file], rule.Name)
				default:
					fmt.Printf("unknown rule type %s", v)
				}
			}
		}
		// get the diff between the two lists of rules
		diff = append(diff, difference(rules, actualRules[file])...)
	}
	// output the missing rules
	if len(diff) != 0 {
		t.Fatalf("Missing alerts: %v:", diff)
	}
}

func execToPod(command string, podname string, namespace string, container string, ctx *TestingContext) (string, error) {
	req := ctx.KubeClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podname).
		Namespace(namespace).
		SubResource("exec").
		Param("container", container)
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		return "", fmt.Errorf("error adding to scheme: %v", err)
	}
	parameterCodec := runtime.NewParameterCodec(scheme)
	req.VersionedParams(&corev1.PodExecOptions{
		Container: container,
		Command:   strings.Fields(command),
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}, parameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(ctx.KubeConfig, "POST", req.URL())
	if err != nil {
		return "", fmt.Errorf("error while creating Executor: %v", err)
	}

	var stdout, stderr bytes.Buffer
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})
	if err != nil {
		return "", fmt.Errorf("error in Stream: %v", err)
	}

	return stdout.String(), nil
}

// difference returns the elements in `a` that aren't in `b`.
func difference(a, b []string) []string {
	mb := make(map[string]struct{}, len(b))
	for _, x := range b {
		mb[x] = struct{}{}
	}

	var diff []string
	for _, x := range a {
		if _, found := mb[x]; !found {
			diff = append(diff, x)
		}
	}
	return diff
}
