package backup

import (
	"context"
	"strings"
	"testing"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	apiv1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCronJob(t *testing.T) {
	var (
		cronJobName     = "cronjob-foo"
		namespace       = "test-namespace"
		generateJobName = "job-foo"
	)

	jobTemplate := batchv1beta1.JobTemplateSpec{}

	// Test backup CronJob
	cronJob := &batchv1beta1.CronJob{
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      cronJobName,
		},
		Spec: batchv1beta1.CronJobSpec{
			JobTemplate: jobTemplate,
		},
	}

	client := createMockClientForCronJob(t, cronJob) // Initialize client with mock CronJob
	executor := NewCronJobBackupExecutor(cronJobName, namespace, generateJobName)

	// In a separate goroutine, wait for the job to be created, and set the
	// completion time to signal the job finished successfully. The goroutine
	// must be started before calling `PerformBackup` as it's a blocking call
	// that waits for the backup to finish
	go func() {
		// Get the job created by the backup executor
		var expectedJob *batchv1.Job
		for {
			existingJobs := &batchv1.JobList{}
			client.List(context.TODO(), existingJobs, k8sclient.InNamespace(namespace))

			for _, existingJob := range existingJobs.Items {
				if !strings.HasPrefix(existingJob.Name, generateJobName) {
					continue
				}

				expectedJob = &existingJob
				break
			}

			if expectedJob != nil {
				break
			}
		}

		// Simulate the time it takes to execute the job
		time.Sleep(time.Second * 1)

		// Set the completion time
		expectedJob.Status.CompletionTime = &v1.Time{
			Time: time.Now(),
		}
		client.Status().Update(context.TODO(), expectedJob)
	}()

	// Call `PerformBackup` and assert that no error is returned
	err := executor.PerformBackup(client, time.Second*10)
	if err != nil {
		t.Errorf("Unexpected error running backup from CronJob: %v", err)
	}
}

func TestCronJob_NoCronJob(t *testing.T) {
	var (
		cronJobName     = "cronjob-foo"
		namespace       = "test-namespace"
		generateJobName = "job-foo"
	)

	client := createMockClientForCronJob(t)
	executor := NewCronJobBackupExecutor(cronJobName, namespace, generateJobName)

	err := executor.PerformBackup(client, time.Second*1)
	if err == nil {
		t.Errorf("Expected backup to fail as no CronJob is found")
	}
}

func TestCronJob_FailedJob(t *testing.T) {
	var (
		cronJobName     = "cronjob-foo"
		namespace       = "test-namespace"
		generateJobName = "job-foo"
		errorMessage    = "MOCK FAIL"
	)

	jobTemplate := batchv1beta1.JobTemplateSpec{}

	// Test backup CronJob
	cronJob := &batchv1beta1.CronJob{
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      cronJobName,
		},
		Spec: batchv1beta1.CronJobSpec{
			JobTemplate: jobTemplate,
		},
	}

	client := createMockClientForCronJob(t, cronJob) // Initialize client with mock CronJob
	executor := NewCronJobBackupExecutor(cronJobName, namespace, generateJobName)

	// In a separate goroutine, wait for the job to be created, and set the
	// completion time to signal the job finished with failure.
	// The goroutine must be started before calling `PerformBackup` as it's a
	// blocking call that waits for the backup to finish
	go func() {
		// Get the job created by the backup executor
		var expectedJob *batchv1.Job
		for {
			existingJobs := &batchv1.JobList{}
			client.List(context.TODO(), existingJobs, k8sclient.InNamespace(namespace))

			for _, existingJob := range existingJobs.Items {
				if !strings.HasPrefix(existingJob.Name, generateJobName) {
					continue
				}

				expectedJob = &existingJob
				break
			}

			if expectedJob != nil {
				break
			}
		}

		// Simulate the time it takes to execute the job
		time.Sleep(time.Second * 1)

		// Update the job with a failure status
		expectedJob.Status.Conditions = []batchv1.JobCondition{
			batchv1.JobCondition{
				Type:    batchv1.JobFailed,
				Status:  apiv1.ConditionTrue,
				Message: errorMessage,
			},
		}

		client.Status().Update(context.TODO(), expectedJob)
	}()

	// Call `PerformBackup` and assert that no error is returned
	err := executor.PerformBackup(client, time.Second*10)
	if err == nil {
		t.Error("Expected backup to fail as Job failed")
	}

	// Assert that the error includes the job failure message
	if !strings.Contains(err.Error(), errorMessage) {
		t.Errorf("Expected error to contain Job failure message, got: %s", err.Error())
	}
}

func buildSchemeForCronJob() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	err := batchv1.AddToScheme(scheme)
	err = batchv1beta1.AddToScheme(scheme)
	return scheme, err
}

func createMockClientForCronJob(t *testing.T, initObjects ...runtime.Object) k8sclient.Client {
	scheme, err := buildSchemeForCronJob()
	if err != nil {
		t.Errorf("Error creating testing scheme: %v", err)
	}

	return fake.NewFakeClientWithScheme(scheme, initObjects...)
}
