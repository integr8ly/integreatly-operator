package resources

import (
	"context"
	"fmt"
	k8sTypes "k8s.io/apimachinery/pkg/types"
	"os"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ZoneLabel is the label that specifies the zone where a node is
	ZoneLabel = "topology.kubernetes.io/zone"
	// AntiAffinityRequiredEnvVar is an environment variable that, when set to
	// true, makes the product pod replicas use "required" anti affinity rules
	// by AZ
	AntiAffinityRequiredEnvVar = "FORCE_ZONE_DISTRIBUTION"
)

// MutateMultiAZAntiAffinity returns a PodTemplateMutation that sets the anti
// affinity by AZ on the label labelMatch. It checks if the affinity rule is
// required or not, and sets the required or preferred affinity based on it
func MutateMultiAZAntiAffinity(ctx context.Context, client k8sclient.Client, labelMatch string) PodTemplateMutation {
	isRequired, err := IsAntiAffinityRequired(ctx, client)
	if err != nil {
		isRequired = false
	}

	return func(obj metav1.Object, podTemplate *corev1.PodTemplateSpec) error {
		labels := obj.GetLabels()
		labelValue, ok := labels[labelMatch]
		if !ok {
			return fmt.Errorf("label %s not found in object", labelMatch)
		}

		podTemplate.Spec.Affinity = SelectAntiAffinityForCluster(isRequired, map[string]string{
			labelMatch: labelValue,
		})

		return nil
	}
}

// MultiAZAntiAffinityPreferred returns the affinity configuration to set the
// preferred anti affinity by AZ on the given matchLabels
func MultiAZAntiAffinityPreferred(matchLabels map[string]string) *corev1.Affinity {
	return &corev1.Affinity{
		PodAntiAffinity: &corev1.PodAntiAffinity{

			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
				{
					PodAffinityTerm: corev1.PodAffinityTerm{
						LabelSelector: &v1.LabelSelector{
							MatchLabels: matchLabels,
						},
						TopologyKey: ZoneLabel,
					},
					Weight: 100,
				},
			},
		},
	}
}

// MultiAZAntiAffinityRequired returns the affinity configuration to set the
// required anti affinity by AZ on the given matchLabels
func MultiAZAntiAffinityRequired(matchLabels map[string]string) *corev1.Affinity {
	return &corev1.Affinity{
		PodAntiAffinity: &corev1.PodAntiAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
				{
					LabelSelector: &v1.LabelSelector{
						MatchLabels: matchLabels,
					},
					TopologyKey: ZoneLabel,
				},
			},
		},
	}
}

// SelectAntiAffinityForCluster returns the affinity configuration for a cluster
// given whether the rule is required or preferred
func SelectAntiAffinityForCluster(required bool, matchLabels map[string]string) *corev1.Affinity {
	if required {
		return MultiAZAntiAffinityRequired(matchLabels)
	}

	return MultiAZAntiAffinityPreferred(matchLabels)
}

// IsAntiAffinityRequired checks whether the anti affinity rule must be set
// to required or preferred.
//
// It currently checks the value of the FORCE_ZONE_DISTRIBUTION bool env var
func IsAntiAffinityRequired(_ context.Context, _ k8sclient.Client) (bool, error) {
	envValue, ok := os.LookupEnv(AntiAffinityRequiredEnvVar)
	if !ok {
		return false, nil
	}

	return strconv.ParseBool(envValue)
}

// IsMultiAZCluster checks if the cluster runs in multiple AZs, by retrieving
// the nodes and checking that the `topology.kubernetes.io/zone` label is
// the same across all
func IsMultiAZCluster(ctx context.Context, client k8sclient.Client) (bool, error) {
	// Get the list of nodes
	nodeList := &corev1.NodeList{}
	if err := client.List(ctx, nodeList); err != nil {
		return false, err
	}

	// If there's no nodes, fail
	if len(nodeList.Items) == 0 {
		return false, fmt.Errorf("no nodes found")
	}

	// If there's only one node, directly return false
	if len(nodeList.Items) == 1 {
		return false, nil
	}

	// Get the zone of the first node. In order to be multi AZ there has
	// to be at least one node with a different zone
	firstZone := nodeList.Items[0].Labels[ZoneLabel]

	// Iterate through the tail of the list and check if there's any difference
	// in the zones
	for i := 1; i < len(nodeList.Items); i++ {
		zone := nodeList.Items[i].Labels[ZoneLabel]
		if zone != firstZone {
			return true, nil
		}
	}

	return false, nil
}

type KindNameSpaceName struct{
	*k8sTypes.NamespacedName
	Kind string
}

func (knn KindNameSpaceName) String()string  {
	return fmt.Sprintf("%s/%s/%s",knn.Kind,knn.Namespace,knn.Name)
}

func FindUnbalanced(ctx context.Context,nameSpace string, client k8sclient.Client)([]string, error){
	nodes := &corev1.NodeList{}
	unBalanced:= []string{}
	if err := client.List(ctx, nodes,&k8sclient.ListOptions{}); err != nil{
		return unBalanced,err
	}
	nodesToZone := map[string]string{}
	for _,n := range nodes.Items{
		for _,a := range n.Status.Addresses{
			if a.Type == corev1.NodeInternalIP{
				nodesToZone[a.Address] = n.Labels[ZoneLabel]
				break
			}
		}
	}
	logrus.Infof("nodes to zone %v", nodesToZone)
	balance := map[string][]string{}
	l := &corev1.PodList{}
	if err := client.List(ctx, l, &k8sclient.ListOptions{Namespace:nameSpace}); err != nil{
		return unBalanced,err
	}
	// need to check if there is more than 1 pod of a kind
	podCount := map[string]int{}
	logrus.Info("total pods in ns ", nameSpace, len(l.Items))
	for _,p := range l.Items{
		if p.Status.Phase != "Running"{
			continue
		}

		for _, o := range p.OwnerReferences{
			if *o.Controller {
				knn := &KindNameSpaceName{
					NamespacedName: &k8sTypes.NamespacedName{
						Namespace: nameSpace,
					},
				}

				if o.Kind == "ReplicationController"{
					knn.Name = p.Annotations["openshift.io/deployment-config.name"]
					knn.Kind = "DeploymentConfig"
				}else{
					knn.Name=o.Name
					knn.Kind=o.Kind
				}
				if _, ok := podCount[knn.Name]; !ok{
					podCount[knn.Name]=1
				}else{
					podCount[p.Name] = podCount[p.Name]+1
				}
				if balance[knn.String()] == nil{
					balance[knn.String()]=[]string{}
				}

				balance[knn.String()] = append(balance[knn.String()],nodesToZone[p.Status.HostIP])
				break
			}
		}
	}
	for knn,z := range balance{
		if len(z) == 1 && podCount[knn] > 1{
			// need rebalance
			unBalanced = append(unBalanced, knn)
		}
	}

	return unBalanced,nil
}
