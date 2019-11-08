package e2e

import (
	appsv1 "k8s.io/api/apps/v1"
	bv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	_ "github.com/lib/pq"
)

// returns job template
func ConnectionJob(container []v1.Container, jobName string, namespace string) *bv1.Job {
	return &bv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: namespace,
		},
		Spec: bv1.JobSpec{
			Parallelism:           Int32Ptr(1),
			Completions:           Int32Ptr(1),
			ActiveDeadlineSeconds: Int64Ptr(300),
			BackoffLimit:          Int32Ptr(1),
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: jobName,
				},
				Spec: v1.PodSpec{
					Containers:    container,
					RestartPolicy: v1.RestartPolicyOnFailure,
				},
			},
		},
	}
}

func GetTestDeployment(name string, namespace string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func GetTestPVC(name string, namespace string) *v1.PersistentVolumeClaim {
	return &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func GetTestService(name string, namespace string) *v1.Service {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func Int32Ptr(i int32) *int32 { return &i }

func Int64Ptr(i int64) *int64 { return &i }
