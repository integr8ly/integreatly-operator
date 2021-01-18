package backup

import (
	"context"
	"fmt"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	apiv1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// CronJobBackupExecutor creates backups by creating a Job from a CronJob and
// waiting for its completion
type CronJobBackupExecutor struct {
	CronJobName     string // Name of the CronJob that performs the backup
	Namespace       string // Namespace where the CronJob is (and the job is created)
	JobGenerateName string // Base name for the created Job
}

func NewCronJobBackupExecutor(cronJobName, namespace, jobGenerateName string) BackupExecutor {
	return &CronJobBackupExecutor{
		CronJobName:     cronJobName,
		Namespace:       namespace,
		JobGenerateName: jobGenerateName,
	}
}

func (e *CronJobBackupExecutor) PerformBackup(client k8sclient.Client, timeout time.Duration) error {
	log.Infof("Performing backup by creating Job", l.Fields{"cronJob": e.CronJobName, "ns": e.Namespace})

	// Generate the job name
	jobName := fmt.Sprintf("%s-%s", e.JobGenerateName, time.Now().Format("2006-01-02-150405"))

	// Get the CronJob to run
	cronJob := &batchv1beta1.CronJob{}
	err := client.Get(context.TODO(), types.NamespacedName{
		Name:      e.CronJobName,
		Namespace: e.Namespace,
	}, cronJob)
	if err != nil {
		return fmt.Errorf("Error obtaining CronJob %s in namespace %s: %v", e.CronJobName, e.Namespace, err)
	}

	// Create the Job based on the CronJob spec
	jobTemplate := cronJob.Spec.JobTemplate
	job := &batchv1.Job{
		ObjectMeta: v1.ObjectMeta{
			Namespace: e.Namespace,
			Name:      jobName,
		},
		Spec: jobTemplate.Spec,
	}
	if err := client.Create(context.TODO(), job); err != nil {
		return fmt.Errorf("Error creating Job from CronJob %s in namespace %s: %v",
			e.CronJobName, e.Namespace, err)
	}

	// Query the newly created job until either it finishes, or it times out
	timeStarted := time.Now()
	for {
		if time.Now().After(timeStarted.Add(timeout)) {
			return fmt.Errorf("Timed out when waiting for Job %s to finish", "")
		}

		queryJob := &batchv1.Job{}
		err = client.Get(context.TODO(), types.NamespacedName{Name: jobName, Namespace: e.Namespace}, queryJob)
		if err != nil {
			return fmt.Errorf("Error querying newly created Job %s in namespace %s: %v", "", e.Namespace, err)
		}

		// If the completion time field is set, the job finished succesfully
		if queryJob.Status.CompletionTime != nil {
			return nil
		}

		// Check if the job finished with errors, if it did, return the error
		if err := getJobError(queryJob); err != nil {
			return fmt.Errorf("Error performing backup job: %w", err)
		}
	}
}

func getJobError(job *batchv1.Job) error {
	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobFailed && condition.Status == apiv1.ConditionTrue {
			return fmt.Errorf("job failed: %v", condition.Message)
		}
	}

	return nil
}
