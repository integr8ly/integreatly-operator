package backup

import (
	"context"
	"fmt"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"strings"
	"testing"
	"time"

	"github.com/integr8ly/integreatly-operator/utils"

	"github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1"
	"github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1/types"
	moqClient "github.com/integr8ly/integreatly-operator/pkg/client"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	fakeResourceVersion = "1000"
)

// TestAWSSnapshotPostgres tests that the AWSBackupExecutor successfully creates
// a PostgresSnapshot and waits for it's completion
func TestAWSSnapshotPostgres(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	namespace := "testing-namespaces-operator"
	resourceName := "test-rhoam-postgres"

	client := moqClient.NewSigsClientMoqWithSchemeWithStatusSubresource(scheme, buildTestPostgresSnapshotCr())
	executor := NewAWSBackupExecutor(namespace, resourceName, PostgresSnapshotType)

	go func() {
		var postgresSnapshot *v1alpha1.PostgresSnapshot
		for {
			existingSnapshots := &v1alpha1.PostgresSnapshotList{}
			if err := client.List(context.TODO(), existingSnapshots, k8sclient.InNamespace(namespace)); err != nil {
				continue
			}

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
		if err := client.Status().Update(context.TODO(), postgresSnapshot); err != nil {
			return
		}
	}()

	err = executor.PerformBackup(client, time.Second*10)
	if err != nil {
		t.Errorf("Unexpected error performing postgres backup: %v", err)
	}
}

// TestAWSSnapshotRedis tests that the AWSBackupExecutor succesfully creates
// and waits for the completion of a RedisSnapshot
func TestAWSSnapshotRedis(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	namespace := "testing-namespaces-operator"
	resourceName := "test-rhoam-redis"

	client := moqClient.NewSigsClientMoqWithSchemeWithStatusSubresource(scheme, buildTestRedisSnapshotCR())
	executor := NewAWSBackupExecutor(namespace, resourceName, RedisSnapshotType)

	go func() {
		var redisSnapshot *v1alpha1.RedisSnapshot
		for {
			existingSnapshots := &v1alpha1.RedisSnapshotList{}
			if err := client.List(context.TODO(), existingSnapshots, k8sclient.InNamespace(namespace)); err != nil {
				continue
			}

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
		if err := client.Status().Update(context.TODO(), redisSnapshot); err != nil {
			return
		}
	}()

	err = executor.PerformBackup(client, time.Second*10)
	if err != nil {
		t.Errorf("Unexpected error performing postgres backup: %v", err)
	}
}

// TestAWSSnapshotPostgres_FailedJob tests that the AWSBackupExecutor returns
// an error when a PostgresSnapshot backup fails
func TestAWSSnapshotPostgres_FailedJob(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	namespace := "testing-namespaces-operator"
	resourceName := "test-rhoam-postgres"

	client := moqClient.NewSigsClientMoqWithSchemeWithStatusSubresource(scheme, buildTestPostgresSnapshotCr())
	executor := NewAWSBackupExecutor(namespace, resourceName, PostgresSnapshotType)

	go func() {
		var postgresSnapshot *v1alpha1.PostgresSnapshot
		for {
			existingSnapshots := &v1alpha1.PostgresSnapshotList{}
			if err := client.List(context.TODO(), existingSnapshots, k8sclient.InNamespace(namespace)); err != nil {
				continue
			}

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
		if err := client.Status().Update(context.TODO(), postgresSnapshot); err != nil {
			return
		}
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
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	namespace := "testing-namespaces-operator"
	resourceName := "test-rhoam-redis"

	client := moqClient.NewSigsClientMoqWithSchemeWithStatusSubresource(scheme, buildTestRedisSnapshotCR())
	executor := NewAWSBackupExecutor(namespace, resourceName, RedisSnapshotType)

	go func() {
		var redisSnapshot *v1alpha1.RedisSnapshot
		for {
			existingSnapshots := &v1alpha1.RedisSnapshotList{}
			if err := client.List(context.TODO(), existingSnapshots, k8sclient.InNamespace(namespace)); err != nil {
				continue
			}

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
		if err := client.Status().Update(context.TODO(), redisSnapshot); err != nil {
			return
		}
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

func buildTestPostgresSnapshotCr() *v1alpha1.PostgresSnapshot {
	return &v1alpha1.PostgresSnapshot{
		ObjectMeta: controllerruntime.ObjectMeta{
			Name:            "test",
			Namespace:       "test",
			ResourceVersion: fakeResourceVersion,
		},
	}
}

func buildTestRedisSnapshotCR() *v1alpha1.RedisSnapshot {
	return &v1alpha1.RedisSnapshot{
		ObjectMeta: controllerruntime.ObjectMeta{
			Name:            "test",
			Namespace:       "test",
			ResourceVersion: fakeResourceVersion,
		},
	}
}
