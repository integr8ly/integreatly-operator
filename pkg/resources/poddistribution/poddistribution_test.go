package poddistribution

import (
	"context"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	apiappsv1 "github.com/openshift/api/apps/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

func getPod(name string, ownerName string, ip string, ownerKind string) corev1.Pod {
	return corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: "redhat-rhoam-3scale",
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind:       ownerKind,
					Name:       ownerName,
					Controller: newTrue(),
				},
			},
			Annotations: map[string]string{
				"openshift.io/deployment-config.name": ownerName,
			},
		},
		Status: corev1.PodStatus{
			Phase:  "Running",
			HostIP: ip,
		},
	}
}

func getNode(name string, zone string, ip string) corev1.Node {
	return corev1.Node{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: "redhat-rhoam-3scale",
			Labels: map[string]string{
				"topology.kubernetes.io/zone": zone,
			},
		},
		Status: corev1.NodeStatus{
			Addresses: []corev1.NodeAddress{
				{
					Type:    "InternalIP",
					Address: ip,
				},
			},
		},
	}
}

func getBuildScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	err := corev1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = appsv1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = apiappsv1.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	return scheme, err
}

func newTrue() *bool {
	b := true
	return &b
}

func TestPodDistribution(t *testing.T) {
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatal(err)
	}
	if err != nil {
		t.Fatal(err)
	}

	dc1 := &apiappsv1.DeploymentConfig{
		ObjectMeta: v1.ObjectMeta{
			Name:      "dc1",
			Namespace: "redhat-rhoam-3scale",
		},
	}

	rs1 := &appsv1.ReplicaSet{
		ObjectMeta: v1.ObjectMeta{
			Name:      "rs1",
			Namespace: "redhat-rhoam-3scale",
		},
	}

	ss1 := &appsv1.StatefulSet{
		ObjectMeta: v1.ObjectMeta{
			Name:      "ss1",
			Namespace: "redhat-rhoam-3scale",
		},
	}

	dc2 := &apiappsv1.DeploymentConfig{
		ObjectMeta: v1.ObjectMeta{
			Name:      "dc2",
			Namespace: "redhat-rhoam-3scale",
			Annotations: map[string]string{
				"pod-balance-attempts": "3",
			},
		},
	}

	rs2 := &appsv1.ReplicaSet{
		ObjectMeta: v1.ObjectMeta{
			Name:      "rs2",
			Namespace: "redhat-rhoam-3scale",
			Annotations: map[string]string{
				"pod-balance-attempts": "3",
			},
		},
	}

	ss2 := &appsv1.StatefulSet{
		ObjectMeta: v1.ObjectMeta{
			Name:      "ss2",
			Namespace: "redhat-rhoam-3scale",
			Annotations: map[string]string{
				"pod-balance-attempts": "3",
			},
		},
	}

	dc3 := &apiappsv1.DeploymentConfig{
		ObjectMeta: v1.ObjectMeta{
			Name:      "dc3",
			Namespace: "invalid-namespace",
			Annotations: map[string]string{
				PodRebalanceAttempts: "3",
			},
		},
	}

	rs3 := &appsv1.ReplicaSet{
		ObjectMeta: v1.ObjectMeta{
			Name:      "rs3",
			Namespace: "redhat-rhoam-3scale",
			Annotations: map[string]string{
				PodRebalanceAttempts: "should-be-int",
			},
		},
	}

	ss3 := &appsv1.StatefulSet{
		ObjectMeta: v1.ObjectMeta{
			Name:      "ss3",
			Namespace: "invalid-namespace",
			Annotations: map[string]string{
				PodRebalanceAttempts: "3",
			},
		},
	}

	podList3 := &corev1.PodList{
		Items: []corev1.Pod{
			getPod("deploypod1", "dc3", "1.1.1.1", "ReplicationController"),
			getPod("deploypod2", "dc3", "1.1.1.1", "ReplicationController"),
			getPod("sspod1", "ss3", "1.1.1.1", "StatefulSet"),
			getPod("sspod2", "ss3", "1.1.1.1", "StatefulSet"),
			getPod("rspod1", "rs3", "1.1.1.1", "ReplicaSet"),
			getPod("rspod2", "rs3", "1.1.1.1", "ReplicaSet"),
		},
	}

	podList1 := &corev1.PodList{
		Items: []corev1.Pod{
			getPod("deploypod1", "dc1", "1.1.1.1", "ReplicationController"),
			getPod("deploypod2", "dc1", "1.1.1.1", "ReplicationController"),
			getPod("sspod1", "ss1", "1.1.1.1", "StatefulSet"),
			getPod("sspod2", "ss1", "1.1.1.1", "StatefulSet"),
			getPod("rspod1", "rs1", "1.1.1.1", "ReplicaSet"),
			getPod("rspod2", "rs1", "1.1.1.1", "ReplicaSet"),
		},
	}

	nodeList1 := &corev1.NodeList{
		Items: []corev1.Node{
			getNode("node1", "zone1", "1.1.1.1"),
			getNode("node2", "zone2", "2.2.2.2"),
		},
	}

	podList2 := &corev1.PodList{
		Items: []corev1.Pod{
			getPod("deploypod1", "dc2", "1.1.1.1", "ReplicationController"),
			getPod("deploypod2", "dc2", "2.2.2.2", "ReplicationController"),
			getPod("sspod1", "ss2", "1.1.1.1", "StatefulSet"),
			getPod("sspod2", "ss2", "2.2.2.2", "StatefulSet"),
			getPod("rspod1", "rs2", "1.1.1.1", "ReplicaSet"),
			getPod("rspod2", "rs2", "2.2.2.2", "ReplicaSet"),
		},
	}

	deleteCount1 := 0
	updateCount1 := 0
	deleteCount2 := 0
	updateCount2 := 0
	deleteCount3 := 0
	updateCount3 := 0

	cases := []struct {
		Name       string
		FakeClient func() k8sclient.Client
		Validate   func(*resources.MultiErr) error
	}{
		{
			Name: "Test pods are forced to distribute",
			FakeClient: func() k8sclient.Client {
				mockClient := moqclient.NewSigsClientMoqWithScheme(scheme, nodeList1, podList1, dc1, rs1, ss1)
				mockClient.DeleteFunc = func(ctx context.Context, obj runtime.Object, opts ...client.DeleteOption) error {
					deleteCount1++
					return nil
				}
				mockClient.UpdateFunc = func(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error {
					updateCount1++
					return nil
				}
				return mockClient
			},
			Validate: func(error *resources.MultiErr) error {
				if deleteCount1 != 3 {
					t.Fatalf("Expected deleteCount of 3, got %d", deleteCount1)
				}
				if updateCount1 != 3 {
					t.Fatalf("Expected updateCount of 3, got %d", updateCount1)
				}
				return nil
			},
		},
		{
			Name: "Test no distribution as pods are correctly distributed",
			FakeClient: func() k8sclient.Client {
				mockClient := moqclient.NewSigsClientMoqWithScheme(scheme, nodeList1, podList2, dc2, rs2, ss2)
				mockClient.DeleteFunc = func(ctx context.Context, obj runtime.Object, opts ...client.DeleteOption) error {
					deleteCount2++
					return nil
				}
				mockClient.UpdateFunc = func(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error {
					updateCount2++
					return nil
				}
				return mockClient
			},
			Validate: func(error *resources.MultiErr) error {
				if deleteCount2 != 0 {
					t.Fatalf("Expected deleteCount of 0, got %d", deleteCount2)
				}
				if updateCount2 != 0 {
					t.Fatalf("Expected updateCount of 0, got %d", updateCount2)
				}
				return nil
			},
		},
		{
			// Even though the pods are not distributed correctly the limit of attempts is reached
			Name: "Test no distribution as limits are reached",
			FakeClient: func() k8sclient.Client {
				mockClient := moqclient.NewSigsClientMoqWithScheme(scheme, nodeList1, podList2, dc2, rs2, ss2)
				mockClient.DeleteFunc = func(ctx context.Context, obj runtime.Object, opts ...client.DeleteOption) error {
					deleteCount3++
					return nil
				}
				mockClient.UpdateFunc = func(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error {
					updateCount3++
					return nil
				}
				return mockClient
			},
			Validate: func(error *resources.MultiErr) error {
				if deleteCount3 != 0 {
					t.Fatalf("Expected deleteCount of 0, got %d", deleteCount3)
				}
				if updateCount3 != 0 {
					t.Fatalf("Expected updateCount of 0, got %d", updateCount3)
				}
				return nil
			},
		},
		{
			Name: "Test that errors are aggregated and returned",
			FakeClient: func() k8sclient.Client {
				mockClient := moqclient.NewSigsClientMoqWithScheme(scheme, nodeList1, podList3, dc3, rs3, ss3)
				return mockClient
			},
			Validate: func(error *resources.MultiErr) error {
				if len(error.Errors) != 3 {
					t.Fatal("Expected 3 errors, got ", len(error.Errors))
				}
				return nil
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			error := ReconcilePodDistribution(context.TODO(), tc.FakeClient(), "redhat-rhoam-", "managed-api")
			if err = tc.Validate(error); err != nil {
				t.Fatal("test validation failed: ", err)
			}
		})
	}
}
