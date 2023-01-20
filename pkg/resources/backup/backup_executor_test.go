package backup

import (
	"fmt"
	"testing"
	"time"

	"github.com/integr8ly/integreatly-operator/test/utils"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestConcurrentBackup(t *testing.T) {
	scheme := runtime.NewScheme()
	client := utils.NewTestClient(scheme)

	// 7 concurrent backups that take 1 second each. Should still take approximately
	// 1 second as they're concurrent
	executor := NewConcurrentBackupExecutor(
		mockBackupExecutor{1 * time.Second},
		mockBackupExecutor{1 * time.Second},
		mockBackupExecutor{1 * time.Second},
		mockBackupExecutor{1 * time.Second},
		mockBackupExecutor{1 * time.Second},
		mockBackupExecutor{1 * time.Second},
		mockBackupExecutor{1 * time.Second},
	)

	timeStarted := time.Now()
	err := executor.PerformBackup(client, time.Second*3)
	timeFinished := time.Now()

	if err != nil {
		t.Errorf("Unexpected error performing concurrent backups: %v", err)
	}

	elapsed := timeFinished.Sub(timeStarted)
	// Add 2 seconds threshold (more than enough) for context switching. If it
	// took more than that the concurrency is not properly implemented
	if elapsed > time.Second*3 {
		t.Errorf("Concurrent backups took too long: %v", elapsed)
	}
}

type mockBackupExecutor struct {
	SleepTime time.Duration
}

func (e mockBackupExecutor) PerformBackup(client k8sclient.Client, timeout time.Duration) error {
	if e.SleepTime > timeout {
		return fmt.Errorf("SleepTime %v for mock is greater than given timeout %v", e.SleepTime, timeout)
	}
	time.Sleep(e.SleepTime)
	return nil
}
