package backup

import (
	"context"
	"fmt"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"time"

	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	crotypes "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var log = l.NewLoggerWithContext(l.Fields{l.ComponentLogContext: "backup"})

// AWSBackupExecutor knows how to perform backups by creating snapshot CRs
// and waiting for their completion
type AWSBackupExecutor struct {
	SnapshotNamespace string          // Namespace where the snapshot CR is created
	ResourceName      string          // AWS Resource name
	SnapshotType      AWSSnapshotType // Type of snapshot CR to create
}

func NewAWSBackupExecutor(snapshotNamespace, resourceName string, snapshotType AWSSnapshotType) BackupExecutor {
	return &AWSBackupExecutor{
		SnapshotNamespace: snapshotNamespace,
		ResourceName:      resourceName,
		SnapshotType:      snapshotType,
	}
}

// AWSSnapshotType represents the type of snapshot to create
type AWSSnapshotType string

const (
	// PostgresSnapshotType creates PostgresSnapshot CRs
	PostgresSnapshotType AWSSnapshotType = "PostgresSnapshot"
	// RedisSnapshotType creates RedisSnapshot CRs
	RedisSnapshotType AWSSnapshotType = "RedisSnapshot"
)

// PerformBackup creates a snapshot CR and waits until the status of the CR
// is `complete`
func (e *AWSBackupExecutor) PerformBackup(client k8sclient.Client, timeout time.Duration) error {
	log.Infof("Performing backup on AWS", l.Fields{"snapshotType": e.SnapshotType, "resourceName": e.ResourceName})

	snapshotName := fmt.Sprintf("%s-preupgrade-snapshot-%s", e.ResourceName, time.Now().Format("2006-01-02-150405"))

	// Initialize the snapshot CR based on the snapshot type
	var snapshotCR runtime.Object
	commonObjectMeta := v1.ObjectMeta{
		Namespace: e.SnapshotNamespace,
		Name:      snapshotName,
	}

	switch e.SnapshotType {
	case PostgresSnapshotType:
		snapshotCR = &v1alpha1.PostgresSnapshot{
			ObjectMeta: commonObjectMeta,
			Spec: v1alpha1.PostgresSnapshotSpec{
				ResourceName: e.ResourceName,
			},
		}
	case RedisSnapshotType:
		snapshotCR = &v1alpha1.RedisSnapshot{
			ObjectMeta: commonObjectMeta,
			Spec: v1alpha1.RedisSnapshotSpec{
				ResourceName: e.ResourceName,
			},
		}
	default:
		return fmt.Errorf("Unsupported value for AWSShapshotType. Expected %s or %s, got %s",
			PostgresSnapshotType, RedisSnapshotType, e.SnapshotType)
	}

	// Create the CR
	err := client.Create(context.TODO(), snapshotCR)
	if err != nil {
		return fmt.Errorf("Error creating %s for backup of resource %s: %v",
			e.SnapshotType, e.ResourceName, err)
	}

	// Initialize the CR to query it's completion
	var queryCR runtime.Object
	switch e.SnapshotType {
	case PostgresSnapshotType:
		queryCR = &v1alpha1.PostgresSnapshot{}
	case RedisSnapshotType:
		queryCR = &v1alpha1.RedisSnapshot{}
	}

	// Request the CR status until it's complete or it times out
	started := time.Now()
	for {
		// If it times out, return an error
		if time.Now().After(started.Add(timeout)) {
			return fmt.Errorf("Snapshot of %s %s timed out", e.ResourceName, e.SnapshotType)
		}

		// Get the CR
		err = client.Get(context.TODO(), types.NamespacedName{
			Name:      snapshotName,
			Namespace: e.SnapshotNamespace,
		}, queryCR)
		if err != nil {
			return fmt.Errorf("Error occurred querying snapshot for backup %s", e.ResourceName)
		}

		// Get the phase
		var phase crotypes.StatusPhase
		var message crotypes.StatusMessage
		switch e.SnapshotType {
		case PostgresSnapshotType:
			typedSnapshotCR := queryCR.(*v1alpha1.PostgresSnapshot)
			phase = typedSnapshotCR.Status.Phase
			message = typedSnapshotCR.Status.Message
		case RedisSnapshotType:
			typedSnapshotCR := queryCR.(*v1alpha1.RedisSnapshot)
			phase = typedSnapshotCR.Status.Phase
			message = typedSnapshotCR.Status.Message
		}

		// If the snapshot failed, return an error with the message
		if phase == crotypes.PhaseFailed {
			return fmt.Errorf("Snapshot failed: %s", message)
		}

		// If it's complete, break the loop
		if phase == crotypes.PhaseComplete {
			break
		}
	}

	return nil
}
