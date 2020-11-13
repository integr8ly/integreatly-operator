package backup

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// TestAWSSnapshotPostgres tests that the AWSBackupExecutor successfully creates
// a PostgresSnapshot and waits for it's completion
func TestAWSSnapshotPostgres(t *testing.T) {
	scheme, err := buildSchemeForAWSBackup()
	if err != nil {
		t.Errorf("Error building scheme: %w", err)
		return
	}

	namespace := "testing-namespaces-operator"
	resourceName := "test-rhmi-postgres"

	client := fake.NewFakeClientWithScheme(scheme)
	executor := NewAWSBackupExecutor(namespace, resourceName, PostgresSnapshotType)

	go func() {
		var postgresSnapshot *v1alpha1.PostgresSnapshot
		for {
			existingSnapshots := &v1alpha1.PostgresSnapshotList{}
			client.List(context.TODO(), existingSnapshots, k8sclient.InNamespace(namespace))

			for _, existingSnapshot := range existingSnapshots.Items {
				if !strings.HasPrefix(existingSnapshot.Name, fmt.Sprintf("%s-preupgrade-snapshot", resourceName)) {
					continue
				}

				postgresSnapshot = &existingSnapshot
				break
			}

			if postgresSnapshot != nil {
				break
			}
		}

		// Simulate the time it would take to finish the backup
		time.Sleep(time.Second * 1)

		postgresSnapshot.Status.Phase = types.PhaseComplete
		client.Status().Update(context.TODO(), postgresSnapshot)
	}()

	err = executor.PerformBackup(client, time.Second*10)
	if err != nil {
		t.Errorf("Unexpected error performing postgres backup: %w", err)
	}
}

// TestAWSSnapshotRedis tests that the AWSBackupExecutor succesfully creates
// and waits for the completion of a RedisSnapshot
func TestAWSSnapshotRedis(t *testing.T) {
	scheme, err := buildSchemeForAWSBackup()
	if err != nil {
		t.Errorf("Error building scheme: %w", err)
		return
	}

	namespace := "testing-namespaces-operator"
	resourceName := "test-rhmi-redis"

	client := fake.NewFakeClientWithScheme(scheme)
	executor := NewAWSBackupExecutor(namespace, resourceName, RedisSnapshotType)

	go func() {
		var redisSnapshot *v1alpha1.RedisSnapshot
		for {
			existingSnapshots := &v1alpha1.RedisSnapshotList{}
			client.List(context.TODO(), existingSnapshots, k8sclient.InNamespace(namespace))

			for _, existingSnapshot := range existingSnapshots.Items {
				if !strings.HasPrefix(existingSnapshot.Name, fmt.Sprintf("%s-preupgrade-snapshot", resourceName)) {
					continue
				}

				redisSnapshot = &existingSnapshot
				break
			}

			if redisSnapshot != nil {
				break
			}
		}

		// Simulate the time it would take to finish the backup
		time.Sleep(time.Second * 1)

		redisSnapshot.Status.Phase = types.PhaseComplete
		client.Status().Update(context.TODO(), redisSnapshot)
	}()

	err = executor.PerformBackup(client, time.Second*10)
	if err != nil {
		t.Errorf("Unexpected error performing postgres backup: %w", err)
	}
}

// TestAWSSnapshotPostgres_FailedJob tests that the AWSBackupExecutor returns
// an error when a PostgresSnapshot backup fails
func TestAWSSnapshotPostgres_FailedJob(t *testing.T) {
	scheme, err := buildSchemeForAWSBackup()
	if err != nil {
		t.Fatalf("Error building scheme: %v", err)
		return
	}

	namespace := "testing-namespaces-operator"
	resourceName := "test-rhmi-postgres"

	client := fake.NewFakeClientWithScheme(scheme)
	executor := NewAWSBackupExecutor(namespace, resourceName, PostgresSnapshotType)

	go func() {
		var postgresSnapshot *v1alpha1.PostgresSnapshot
		for {
			existingSnapshots := &v1alpha1.PostgresSnapshotList{}
			client.List(context.TODO(), existingSnapshots, k8sclient.InNamespace(namespace))

			for _, existingSnapshot := range existingSnapshots.Items {
				if !strings.HasPrefix(existingSnapshot.Name, fmt.Sprintf("%s-preupgrade-snapshot", resourceName)) {
					continue
				}

				postgresSnapshot = &existingSnapshot
				break
			}

			if postgresSnapshot != nil {
				break
			}
		}

		// Simulate the time it would take to finish the backup
		time.Sleep(time.Second * 1)

		// Set a failed status
		postgresSnapshot.Status.Phase = types.PhaseFailed
		postgresSnapshot.Status.Message = "MOCK FAIL"
		client.Status().Update(context.TODO(), postgresSnapshot)
	}()

	err = executor.PerformBackup(client, time.Second*10)
	if err == nil {
		t.Fatal("Expected error when performing fail backup")
		return
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "MOCK FAIL") {
		t.Errorf("Expected error message to contain PostgresSnapshot status message, but got %s", errMsg)
	}
}

// TestAWSSnapshotRedis_FailedJob tests that the AWSBackupExecutor returns
// an error when a RedisSnapshot backup fails
func TestAWSSnapshotRedis_FailedJob(t *testing.T) {
	scheme, err := buildSchemeForAWSBackup()
	if err != nil {
		t.Fatalf("Error building scheme: %v", err)
		return
	}

	namespace := "testing-namespaces-operator"
	resourceName := "test-rhmi-redis"

	client := fake.NewFakeClientWithScheme(scheme)
	executor := NewAWSBackupExecutor(namespace, resourceName, RedisSnapshotType)

	go func() {
		var redisSnapshot *v1alpha1.RedisSnapshot
		for {
			existingSnapshots := &v1alpha1.RedisSnapshotList{}
			client.List(context.TODO(), existingSnapshots, k8sclient.InNamespace(namespace))

			for _, existingSnapshot := range existingSnapshots.Items {
				if !strings.HasPrefix(existingSnapshot.Name, fmt.Sprintf("%s-preupgrade-snapshot", resourceName)) {
					continue
				}

				redisSnapshot = &existingSnapshot
				break
			}

			if redisSnapshot != nil {
				break
			}
		}

		// Simulate the time it would take to finish the backup
		time.Sleep(time.Second * 1)

		// Set a failed status
		redisSnapshot.Status.Phase = types.PhaseFailed
		redisSnapshot.Status.Message = "MOCK FAIL"
		client.Status().Update(context.TODO(), redisSnapshot)
	}()

	err = executor.PerformBackup(client, time.Second*10)
	if err == nil {
		t.Fatal("Expected error when performing fail backup")
		return
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "MOCK FAIL") {
		t.Errorf("Expected error message to contain RedisSnapshot status message, but got %s", errMsg)
	}
}

func buildSchemeForAWSBackup() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	err := v1alpha1.SchemeBuilder.AddToScheme(scheme)

	return scheme, err
}
