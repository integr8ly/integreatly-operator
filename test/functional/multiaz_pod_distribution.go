package functional

import (
	goctx "context"
	"fmt"
	"github.com/integr8ly/integreatly-operator/test/common"
	v1 "k8s.io/api/core/v1"
	"math"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"testing"
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
}

func TestMultiAZPodDistribution(t *testing.T, ctx *common.TestingContext) {
	pods := &v1.PodList{}
	nodes := &v1.NodeList{}

	ctx.Client.List(goctx.TODO(), nodes)

	availableZones = getAZ(nodes)

	for _, ns := range namespacesToCheck {
		distributionPerOwner := make(map[string]*podDistribution)
		// Get a list of pods per namespace
		ctx.Client.List(goctx.TODO(), pods, k8sclient.InNamespace(ns))
		for _, pod := range pods.Items {
			// Skip pods with status "Completed"
			if pod.Status.Phase == "Succeeded" {
				continue
			}
			// Get an owner of the pod (ReplicationController, StatefulSet, ...)
			podOwnerName := pod.OwnerReferences[0].Name
			// Get a pod's AZ it is currently running in
			podsAz := getPodZone(pod, nodes)
			// Save
			if _, ok := distributionPerOwner[podOwnerName]; ok == false {
				distributionPerOwner[podOwnerName] = &podDistribution{zones: map[string]int{podsAz: 1}, podsTotal: 1}
			} else {
				distributionPerOwner[podOwnerName].podsTotal++
				distributionPerOwner[podOwnerName].zones[podsAz]++
			}

		}
		testErrors := testCorrectPodDistribution(distributionPerOwner)

		if len(testErrors) != 0 {
			t.Fatalf("Error when verifying the pod distribution: %s", testErrors)
		}
	}
}

func getAZ(nodes *v1.NodeList) map[string]bool {
	zones := make(map[string]bool)
	for _, node := range nodes.Items {
		if isNodeWorkerAndReady(node) {
			for labelName, labelValue := range node.Labels {
				if labelName == "topology.kubernetes.io/zone" {
					zones[labelValue] = true
				}
			}
		}
	}
	return zones
}

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

func getPodZone(pod v1.Pod, nodes *v1.NodeList) string {
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

func testCorrectPodDistribution(dist map[string]*podDistribution) []string {
	var testErrors []string
	for podOwner, pd := range dist {
		minPodsPerZone, maxPodsPerZone := getAllowedNumberOfPodsPerZone(pd.podsTotal)
		for _, n := range pd.zones {
			if n == minPodsPerZone || n == maxPodsPerZone {
				continue
			}
			testErrors = append(testErrors, fmt.Sprintf("Pods are not distributed correctly in %s.", podOwner))
		}
	}
	return testErrors
}

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
