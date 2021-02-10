package poddistribution

import (
	"context"
	"fmt"
	"strconv"
	"time"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	appsv1 "github.com/openshift/api/apps/v1"
	"github.com/sirupsen/logrus"
	k8appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sTypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ZoneLabel is the label that specifies the zone where a node is
	ZoneLabel = "topology.kubernetes.io/zone"
	// Annotation counter on the pods controller, dc, rs, ss
	PodRebalanceAttempts = "pod-rebalance-attempts"
	maxBalanceAttempts   = 3
)

type KindNameSpaceName struct {
	*k8sTypes.NamespacedName
	Obj  runtime.Object
	Kind string
}

func (knn KindNameSpaceName) String() string {
	return fmt.Sprintf("%s/%s/%s", knn.Kind, knn.Namespace, knn.Name)
}

// Check the PodBalanceAttempts, ensure less than maxBalanceAttempts
func verifyRebalanceCount(ctx context.Context, client k8sclient.Client, obj runtime.Object) (bool, error) {
	metaObj, err := meta.Accessor(obj)
	if err != nil {
		return false, nil
	}
	ant := metaObj.GetAnnotations()
	// If there are no annotations then 0 balance attempts have been made
	if ant == nil {
		return true, nil
	}
	if val, ok := ant[PodRebalanceAttempts]; ok {
		i, err := strconv.Atoi(val)
		if err != nil {
			return false, fmt.Errorf("Error converting string annotations %s", ant[PodRebalanceAttempts])
		} else {
			if i >= maxBalanceAttempts {
				logrus.Warningf("Reached max balance attempts for %s on %s", metaObj.GetName(), metaObj.GetNamespace())
				return false, nil
			}
		}
	}

	return true, nil
}

func getObject(ctx context.Context, client k8sclient.Client, knn *KindNameSpaceName) (runtime.Object, error) {
	object := knn.Obj
	err := client.Get(ctx, k8sTypes.NamespacedName{Name: knn.Name, Namespace: knn.Namespace}, object)
	if err != nil {
		return nil, fmt.Errorf("Error getting object %s, on ns %s: %w", knn.Name, knn.Namespace, err)
	}
	return object, nil

}

func getNamespaces(nsPrefix string, installType string) []string {
	namespaces := []string{
		nsPrefix + "3scale",
		nsPrefix + "rhsso",
		nsPrefix + "user-sso",
	}

	if installType == string(integreatlyv1alpha1.InstallationTypeManagedApi) {
		namespaces = append(namespaces, nsPrefix+"marin3r")
	}
	return namespaces
}

func ReconcilePodDistribution(ctx context.Context, client k8sclient.Client, nsPrefix string, installType string) *resources.MultiErr {
	var mErr = &resources.MultiErr{}

	isMultiAZCluster, err := resources.IsMultiAZCluster(ctx, client)
	if err != nil {
		mErr.Add(err)
		return mErr
	}
	if !isMultiAZCluster {
		return mErr
	}

	for _, ns := range getNamespaces(nsPrefix, installType) {
		logrus.Infof("Reconciling Pod Balance in ns %s", ns)
		unbalanced, err := findUnbalanced(ctx, ns, client)
		if err != nil {
			mErr.Add(fmt.Errorf("Error getting pods to balance on namespace %s. %w", ns, err))
			continue
		}
		for knn, pods := range unbalanced {
			obj, err := getObject(ctx, client, knn)
			if err != nil {
				mErr.Add(err)
				continue
			}
			rebalance, err := verifyRebalanceCount(ctx, client, obj)
			if err != nil {
				mErr.Add(err)
				continue
			}
			if rebalance {
				err := forceRebalance(ctx, client, knn, pods)
				if err != nil {
					mErr.Add(err)
				}
			}
		}
	}
	return mErr
}

func findUnbalanced(ctx context.Context, nameSpace string, client k8sclient.Client) (map[*KindNameSpaceName][]string, error) {
	nodes := &corev1.NodeList{}
	unBalanced := map[*KindNameSpaceName][]string{}
	if err := client.List(ctx,
		nodes, &k8sclient.ListOptions{}); err != nil {
		return unBalanced, err
	}
	nodesToZone := map[string]string{}
	for _, n := range nodes.Items {
		for _, a := range n.Status.Addresses {
			if a.Type == corev1.NodeInternalIP {
				nodesToZone[a.Address] = n.Labels[ZoneLabel]
				break
			}
		}
	}
	logrus.Debugf("nodes to zone %v", nodesToZone)
	allKnn := []*KindNameSpaceName{}
	balance := map[*KindNameSpaceName][]string{}
	objPods := map[*KindNameSpaceName][]string{}
	l := &corev1.PodList{}
	listOpts := []k8sclient.ListOption{
		k8sclient.InNamespace(nameSpace),
	}
	if err := client.List(ctx, l, listOpts...); err != nil {
		return unBalanced, fmt.Errorf("Error getting pod lists %w", err)
	}

	// need to check if there is more than 1 pod of a kind
	podCount := map[*KindNameSpaceName]int{}
	logrus.Debugf("total pods in ns %s: %d", nameSpace, len(l.Items))
	for _, p := range l.Items {
		if p.Status.Phase != "Running" {
			continue
		}

		for _, o := range p.OwnerReferences {
			if o.Controller != nil && *o.Controller {
				knn := &KindNameSpaceName{
					NamespacedName: &k8sTypes.NamespacedName{
						Namespace: nameSpace,
					},
				}

				if o.Kind == "ReplicationController" {
					knn.Name = p.Annotations["openshift.io/deployment-config.name"]
					knn.Obj = &appsv1.DeploymentConfig{}
					knn.Kind = "dc"
				} else if o.Kind == "StatefulSet" {
					knn.Name = o.Name
					knn.Obj = &k8appsv1.StatefulSet{}
					knn.Kind = "ss"
				} else if o.Kind == "ReplicaSet" {
					knn.Name = o.Name
					knn.Obj = &k8appsv1.ReplicaSet{}
					knn.Kind = "rs"
				}

				// If this knn already exists use it.
				knn, allKnn = getExisting(knn, allKnn)

				if _, ok := podCount[knn]; !ok {
					podCount[knn] = 1
				} else {
					podCount[knn] = podCount[knn] + 1
				}
				if balance[knn] == nil {
					balance[knn] = []string{}
				}

				if objPods[knn] == nil {
					objPods[knn] = []string{}
				}
				objPods[knn] = append(objPods[knn], p.Name)

				zone := nodesToZone[p.Status.HostIP]
				if !zoneExists(balance[knn], zone) {
					balance[knn] = append(balance[knn], nodesToZone[p.Status.HostIP])
				}
				break
			}
		}
	}
	for knn := range balance {
		if len(balance[knn]) == 1 && podCount[knn] > 1 {
			logrus.Warningf("Requires pod rebalance %s", knn)
			unBalanced[knn] = objPods[knn]
		}
	}

	return unBalanced, nil
}

func getExisting(knn *KindNameSpaceName, allKnn []*KindNameSpaceName) (*KindNameSpaceName, []*KindNameSpaceName) {
	for _, val := range allKnn {
		if knn.Namespace == val.Namespace &&
			knn.Name == val.Name &&
			knn.Kind == val.Kind {
			return val, allKnn
		}
	}
	return knn, append(allKnn, knn)
}

func zoneExists(zones []string, zone string) bool {
	for _, z := range zones {
		if z == zone {
			return true
		}
	}
	return false
}

// Delete a single pod to force redistribution
func forceRebalance(ctx context.Context, client k8sclient.Client, knn *KindNameSpaceName, pods []string) error {

	deletePod(ctx, client, pods[0], knn.Namespace)
	// The wait prevents version clash errors when updating the controller
	err := wait.Poll(time.Second*5, time.Second*5, func() (done bool, err error) {
		err = updatePodBalanceAttemptsOnKNN(ctx, client, knn)
		return true, err
	})
	if err != nil {
		return err
	}
	return nil
}

func deletePod(ctx context.Context, client k8sclient.Client, podName string, ns string) {
	logrus.Infof("Attempting to delete pod %s, on ns %s", podName, ns)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: ns,
		},
	}
	err := client.Get(ctx, k8sclient.ObjectKey{
		Name:      podName,
		Namespace: ns,
	}, pod)
	if err != nil {
		logrus.Errorf("Error getting pod %s on namespace %s. %v", podName, ns, err)
		return
	}
	if err := client.Delete(ctx, pod); err != nil {
		logrus.Errorf("Error deleting pod %s on namespace %s. %v", podName, ns, err)
	}
}

func updatePodBalanceAttemptsOnKNN(ctx context.Context, client k8sclient.Client, knn *KindNameSpaceName) error {
	obj := knn.Obj

	err := client.Get(ctx, k8sclient.ObjectKey{
		Name:      knn.Name,
		Namespace: knn.Namespace,
	}, obj)

	if err != nil {
		return fmt.Errorf("Error getting %s %s on namespace %s. %w", knn.Kind, knn.Name, knn.Namespace, err)
	}
	metaObj, err := meta.Accessor(obj)
	if err != nil {
		return err
	}
	ant, err := getAnnotations(metaObj.GetAnnotations(), knn.Name, knn.Namespace)
	if err != nil {
		return err
	}
	metaObj.SetAnnotations(ant)
	if err := client.Update(ctx, obj); err != nil {
		return fmt.Errorf("Error Updating %s %s on %s. %w", knn.Kind, knn.Name, knn.Namespace, err)
	}
	logrus.Infof("Successfully updated %s %s on %s", knn.Kind, knn.Name, knn.Namespace)
	return nil
}

func getAnnotations(ant map[string]string, name string, ns string) (map[string]string, error) {
	if ant == nil {
		ant = map[string]string{}
	}
	if val, ok := ant[PodRebalanceAttempts]; ok {
		i, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("Error converting annotation to int %s, %s, %s", PodRebalanceAttempts, name, ns)
		}
		i = i + 1
		ant[PodRebalanceAttempts] = strconv.Itoa(i)
	} else {
		ant[PodRebalanceAttempts] = "1"
	}

	return ant, nil
}
