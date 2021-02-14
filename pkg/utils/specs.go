package utils

import (
	wfv1 "github.com/arunprasadmudaliar/trinity/api/v1"
	batchv1 "k8s.io/api/batch/v1"
	batch "k8s.io/api/batch/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func cronJobSpec(name string, namespace string, schedule string) *batch.CronJob {
	var zero *int32
	zero = new(int32)
	*zero = 0
	return &batch.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "wf-cron-" + name,
			Namespace: namespace,
		},
		Spec: batch.CronJobSpec{
			Schedule:                   schedule,
			FailedJobsHistoryLimit:     zero,
			SuccessfulJobsHistoryLimit: zero,
			JobTemplate: batch.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name:            name,
									Image:           "arunmudaliar/trinity:latest",
									ImagePullPolicy: "Always",
									Command:         []string{"trinity"},
									Args: []string{
										"run",
										"-w", name,
										"-n", namespace,
									},
								},
							},
							RestartPolicy: "Never",
						},
					},
				},
			},
		},
	}
}

func podSpec(name string, namespace string, image string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"workflow": name,
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:    "exec",
					Image:   image,
					Command: []string{"date"},
				},
			},
			RestartPolicy: "Never",
		},
	}
}

func jobSpec(name string, namespace string, image string, runid string, taskid string, creds wfv1.MinioCreds) *batchv1.Job {
	var ttl *int32
	ttl = new(int32)
	*ttl = 0
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-task-" + taskid,
			Namespace: namespace,
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: ttl,
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:            name,
							Image:           image,
							ImagePullPolicy: "Always",
							Command:         []string{"trinity"},
							Env: []v1.EnvVar{
								{
									Name:  "MINIO_ROOT_USER",
									Value: creds.AccessKey,
								},
								{
									Name:  "MINIO_ROOT_PASSWORD",
									Value: creds.SecretKey,
								},
							},
							Args: []string{
								"exec",
								"-w", name,
								"-n", namespace,
								"-r", runid,
								"-t", taskid,
							},
						},
					},
					RestartPolicy: "Never",
				},
			},
		},
	}

}

func minioPodSpec(name string, namespace string, creds wfv1.MinioCreds) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-artifact",
			Namespace: namespace,
			Labels: map[string]string{
				"workflow": name,
				"type":     "artifact",
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:    "minio",
					Image:   "minio/minio",
					Command: []string{"minio"},
					Args:    []string{"server", "/data"},
					Env: []v1.EnvVar{
						{
							Name:  "MINIO_ROOT_USER",
							Value: creds.AccessKey,
						},
						{
							Name:  "MINIO_ROOT_PASSWORD",
							Value: creds.SecretKey,
						},
					},
					Ports: []v1.ContainerPort{{ContainerPort: 9000}},
				},
			},
			RestartPolicy: "Never",
		},
	}
}

func minioSvcSpec(name string, namespace string) *v1.Service {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-artifact-svc",
			Namespace: namespace,
			Labels: map[string]string{
				"workflow": name,
				"type":     "artifact",
			},
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{Port: 80, TargetPort: intstr.Parse("9000")},
			},
			Selector: map[string]string{
				"workflow": name,
				"type":     "artifact",
			},
		},
	}
}
