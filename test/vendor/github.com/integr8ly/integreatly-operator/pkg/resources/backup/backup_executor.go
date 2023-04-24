package backup

import (
	"fmt"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"golang.org/x/sync/errgroup"
	"time"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// BackupExecutor knows how to perform backups and wait for their successful
// completion
type BackupExecutor interface {
	PerformBackup(client k8sclient.Client, timeout time.Duration) error
}

// NoopBackupExecutor does nothing. For components that do not require backups
type NoopBackupExecutor struct{}

func NewNoopBackupExecutor() BackupExecutor {
	return &NoopBackupExecutor{}
}

// PerformBackup simply returns a `nil` error
func (e *NoopBackupExecutor) PerformBackup(client k8sclient.Client, timeout time.Duration) error {
	log.Info("No backup to perform")
	return nil
}

// ConcurrentBackupExecutor performs backups by delegating the operation into
// a list of `BackupExecutor` that are performed concurrently in separate
// goroutines
type ConcurrentBackupExecutor struct {
	Executors []BackupExecutor
}

func NewConcurrentBackupExecutor(executors ...BackupExecutor) BackupExecutor {
	return &ConcurrentBackupExecutor{
		Executors: executors,
	}
}

func (e *ConcurrentBackupExecutor) PerformBackup(client k8sclient.Client, timeout time.Duration) error {
	log.Infof("Concurrently performing backups", l.Fields{"backups": len(e.Executors)})

	var g errgroup.Group

	for _, backup := range e.Executors {
		// We need to re-assign the BackupExecutor instance in the scope of the
		// loop, otherwise, as the goroutine might start in another iteration,
		// the value pointed by the `backup` variable will have changed
		each := backup
		g.Go(func() error {
			return each.PerformBackup(client, timeout)
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("Error occurred when performing concurrent backups: %v", err)
	}

	return nil
}
