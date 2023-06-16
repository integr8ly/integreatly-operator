package status

import (
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ObjectWithAppStatus interface {
	client.Object
	GetStatus() AppStatus
}

// Health not yet implemented
type Health string

const (
	Health_Healthy     Health = "Healthy"
	Health_Progressing Health = "Progressing"
	Health_Degraded    Health = "Degraded"
	Health_Suspended   Health = "Suspended"
	Health_Unknown     Health = "Unknown"
)

type AppStatus interface {
	// GetHealth(types.NamespacedName) Health
	// SetHealth(types.NamespacedName, Health)
	GetDeploymentStatus(types.NamespacedName) *appsv1.DeploymentStatus
	SetDeploymentStatus(types.NamespacedName, *appsv1.DeploymentStatus)
	GetStatefulSetStatus(types.NamespacedName) *appsv1.StatefulSetStatus
	SetStatefulSetStatus(types.NamespacedName, *appsv1.StatefulSetStatus)
}

type UnimplementedDeploymentStatus struct{}

func (u *UnimplementedDeploymentStatus) GetDeployments() []types.NamespacedName {
	return nil
}

func (u *UnimplementedDeploymentStatus) GetDeploymentStatus(types.NamespacedName) *appsv1.DeploymentStatus {
	return nil
}

func (u *UnimplementedDeploymentStatus) SetDeploymentStatus(types.NamespacedName, *appsv1.DeploymentStatus) {
}

type UnimplementedStatefulSetStatus struct{}

func (u *UnimplementedStatefulSetStatus) GetStatefulSets() []types.NamespacedName {
	return nil
}

func (u *UnimplementedStatefulSetStatus) GetStatefulSetStatus(types.NamespacedName) *appsv1.StatefulSetStatus {
	return nil
}

func (u *UnimplementedStatefulSetStatus) SetStatefulSetStatus(types.NamespacedName, *appsv1.StatefulSetStatus) {
}
