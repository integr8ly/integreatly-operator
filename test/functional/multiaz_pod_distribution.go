package functional

import (
	goctx "context"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/integr8ly/integreatly-operator/test/common"
	v1 "k8s.io/api/core/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	namespacesToCheck = []string{
		common.RHSSOUserProductOperatorNamespace,
		common.RHSSOProductNamespace,
		common.ThreeScaleProductNamespace,
	}
	availableZones = map[string]bool{}
)

type podDistribution struct {
	zones     map[string]int
	podsTotal int
	namespace string
}

func TestMultiAZPodDistribution(t common.TestingTB, ctx *common.TestingContext) {
	var testErrors []string
	pods := &v1.PodList{}
	nodes := &v1.NodeList{}

	ctx.Client.List(goctx.TODO(), nodes)

	availableZones = GetClustersAvailableZones(nodes)

	// If "NAMESPACES_TO_CHECK" env var contains a list of namespaces, use that instead of the predefined list
	namespacesFromEnvVar := os.Getenv("NAMESPACES_TO_CHECK")
	if namespacesFromEnvVar != "" {
		namespacesToCheck = strings.Split(namespacesFromEnvVar, ",")
	}

	for _, ns := range namespacesToCheck {
		distributionPerOwner := make(map[string]*podDistribution)
		// Get a list of pods per namespace
		err := ctx.Client.List(goctx.TODO(), pods, k8sclient.InNamespace(ns))
		if err != nil {
			testErrors = append(testErrors, fmt.Sprintf("Can't list pods in the namespace %s, %v", ns, err))
			continue
		}
		if len(pods.Items) == 0 {
			testErrors = append(testErrors, fmt.Sprintf("A namespace '%s' doesn't contain any pods", ns))
			continue
		}
		for _, pod := range pods.Items {
			// Skip pods with status "Completed"
			if pod.Status.Phase == "Succeeded" {
				continue
			}
			// Get an owner of the pod (ReplicationController, StatefulSet, ...)
			podOwnerName := pod.OwnerReferences[0].Name
			// Get a pod's AZ it is currently running in
			podsAz := getPodZoneName(pod, nodes)
			// Update the pod distribution for the pods' owner
			if _, ok := distributionPerOwner[podOwnerName]; ok == false {
				distributionPerOwner[podOwnerName] = &podDistribution{
					zones:     map[string]int{podsAz: 1},
					podsTotal: 1,
					namespace: ns,
				}
			} else {
				distributionPerOwner[podOwnerName].podsTotal++
				distributionPerOwner[podOwnerName].zones[podsAz]++
			}

		}
		testErrors = append(testErrors, testCorrectPodDistribution(distributionPerOwner)...)
	}

	if len(testErrors) != 0 {
		t.Fatalf("\nError when verifying the pod distribution: \n%s", testErrors)
	}
}

// Returns true if the node is a "worker" (compute node)
// and is marked as Ready
func isNodeWorkerAndReady(node v1.Node) bool {
	var isWorker bool

	for annKey, annValue := range node.Annotations {
		if annKey == "machine.openshift.io/machine" && strings.Contains(annValue, "worker") {
			isWorker = true
			break
		}
	}

	if isWorker {
		for _, cond := range node.Status.Conditions {
			if cond.Type == "Ready" && cond.Status == "True" {
				return true
			}
		}
	}
	return false
}

// Returns an AWS zone the pod is currently running in
func getPodZoneName(pod v1.Pod, nodes *v1.NodeList) string {
	for _, node := range nodes.Items {
		if pod.Spec.NodeName == node.Name {
			for labelName, labelValue := range node.Labels {
				if labelName == "topology.kubernetes.io/zone" {
					return labelValue
				}

			}
		}
	}
	return ""
}

// Test whether the pod distribution (per pod owner) is correct
// and return a slice of errors (strings) if any was encountered
// Verify that all pods are not on the same zone with the exception of
// replica count = 1
func testCorrectPodDistribution(dist map[string]*podDistribution) []string {
	var testErrors []string
	for podOwner, pd := range dist {
		if pd.podsTotal > 1 {
			for _, n := range pd.zones {
				if n == pd.podsTotal {
					testErrors = append(testErrors, fmt.Sprintf("Pod owner '%s'. All Pods are on the same zone. %+v\n", podOwner, pd))
					break
				}
			}
		}
	}
	return testErrors

}

// Takes the total number of pods belonging to the pod owner
// and calculates the minimal and maximal amount of pods that should be running
// per zone to meet the criteria for uniform pod distribution across the zones
// Examples:
// - 5 pods, 3 zones => 5/3 => min 1, max 2 pods per zone
// - 4 pods, 2 zones => 4/2 => min 2, max 2 pods per zone
func getAllowedNumberOfPodsPerZone(podsTotal int) (min int, max int) {
	if podsTotal%len(availableZones) == 0 {
		min = podsTotal / len(availableZones)
		max = min
		return
	}
	min = int(math.Floor(float64(podsTotal) / float64(len(availableZones))))
	max = min + 1
	return
}
