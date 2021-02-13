package utils

import (
	"context"
	"io/ioutil"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	batch "k8s.io/api/batch/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

//Client returns a kubernetes client
func Client(configpath string) (*kubernetes.Clientset, error) {

	if configpath == "" {
		logrus.Info("Using Incluster configuration")
		config, err := rest.InClusterConfig()
		if err != nil {
			logrus.Fatalf("Error occured while reading incluster kubeconfig:%v", err)
			return nil, err
		}
		return kubernetes.NewForConfig(config)
	}

	logrus.Infof("Using configuration file:%s", configpath)
	config, err := clientcmd.BuildConfigFromFlags("", configpath)
	if err != nil {
		logrus.Fatalf("Error occured while reading kubeconfig:%v", err)
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

func GetObjectMetaData(obj interface{}) (objectMeta metav1.ObjectMeta) {
	switch object := obj.(type) {
	case *v1.Namespace:
		objectMeta = object.ObjectMeta
	}

	return objectMeta
}

func getCron(kc *kubernetes.Clientset, name string, namespace string) bool {
	_, err := kc.BatchV1beta1().CronJobs(namespace).Get(context.Background(), "wf-cron-"+name, metav1.GetOptions{})
	if err != nil {
		return false
	}
	return true
}

func CreateCron(kc *kubernetes.Clientset, name string, namespace string, schedule string) (bool, error) {
	jobexists := getCron(kc, name, namespace)

	if !jobexists {
		_, err := kc.BatchV1beta1().CronJobs(namespace).Create(context.Background(), cronJobSpec(name, namespace, schedule), metav1.CreateOptions{})
		if err != nil {
			return false, err
		}

		return true, nil
	}

	//logrus.Infof("Created cronjob for workflow:%s", name)
	return false, nil
}

func DeleteCron(kc *kubernetes.Clientset, name string, namespace string) (bool, error) {
	jobexists := getCron(kc, name, namespace)
	if jobexists {
		err := kc.BatchV1beta1().CronJobs(namespace).Delete(context.Background(), "wf-cron-"+name, metav1.DeleteOptions{})
		if err != nil {
			return false, err
		}
		return true, nil
	}

	//logrus.Infof("Deleted cronjob for workflow:%s", name)
	return false, nil
}

func UpdateCron(kc *kubernetes.Clientset, name string, namespace string, schedule string) error {
	_, err := kc.BatchV1beta1().CronJobs(namespace).Update(context.Background(), cronJobSpec(name, namespace, schedule), metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	//logrus.Infof("Updated cronjob for workflow:%s", name)
	return nil
}

func CreatePod(kc *kubernetes.Clientset, name string, namespace string, image string) (*v1.Pod, error) {
	podspec := podSpec(name, namespace, image)
	pod, err := kc.CoreV1().Pods(namespace).Create(context.Background(), podspec, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return pod, nil
}

func WatchPod(kc *kubernetes.Clientset, name string, namespace string) (watch.Interface, error) {
	opts := metav1.ListOptions{
		FieldSelector: "metadata.name=" + name,
	}
	return kc.CoreV1().Pods(namespace).Watch(context.Background(), opts)
}

func CreateJob(kc *kubernetes.Clientset, name string, namespace string, image string, runid string, taskid string) (*batchv1.Job, error) {
	jobspec := jobSpec(name, namespace, image, runid, taskid)
	job, err := kc.BatchV1().Jobs(namespace).Create(context.Background(), jobspec, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return job, nil
}

func DeleteJob(kc *kubernetes.Clientset, name string, namespace string) error {
	err := kc.BatchV1().Jobs(namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
	return err
}

func WatchJob(kc *kubernetes.Clientset, name string, namespace string) (watch.Interface, error) {
	opts := metav1.ListOptions{
		FieldSelector: "metadata.name=" + name,
	}
	return kc.BatchV1().Jobs(namespace).Watch(context.Background(), opts)
}

func DeployMinio(kc *kubernetes.Clientset, name string, namespace string) (*v1.Pod, *v1.Service, error) {
	podspec := minioPodSpec(name, namespace)
	svcspec := minioSvcSpec(name, namespace)
	pod, err := kc.CoreV1().Pods(namespace).Create(context.Background(), podspec, metav1.CreateOptions{})
	if err != nil {
		return nil, nil, err
	}
	svc, err := kc.CoreV1().Services(namespace).Create(context.Background(), svcspec, metav1.CreateOptions{})
	if err != nil {
		return nil, nil, err
	}
	return pod, svc, nil
}

func DeleteMinio(kc *kubernetes.Clientset, pod *v1.Pod, svc *v1.Service) error {
	err := kc.CoreV1().Pods(pod.ObjectMeta.Namespace).Delete(context.Background(), pod.ObjectMeta.Name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	err = kc.CoreV1().Services(svc.ObjectMeta.Namespace).Delete(context.Background(), svc.ObjectMeta.Name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	return nil
}

func UploadArtifacts(bucket string, url string, artifacts []string) error {

	ctx := context.Background()
	endpoint := url
	accessKeyID := "minioadmin"
	secretAccessKey := "minioadmin"
	useSSL := false

	// Initialize minio client object.
	mc, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return err
	}

	err = mc.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
	if err != nil {
		return err
	}

	for _, artifact := range artifacts {
		_, err := mc.FPutObject(ctx, bucket, artifact, "/artifacts/outgoing/"+artifact, minio.PutObjectOptions{})
		if err != nil {
			logrus.WithError(err).Errorf("Failed to upload artifact %s", artifact)
		}

	}

	return nil
}

func DownloadArtifacts(bucket string, url string) error {
	endpoint := url
	accessKeyID := "minioadmin"
	secretAccessKey := "minioadmin"
	useSSL := false

	// Initialize minio client object.
	mc, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	objectCh := mc.ListObjects(ctx, bucket, minio.ListObjectsOptions{})
	for object := range objectCh {
		if object.Err != nil {
			//logrus.WithError(err).Error("failed to list artifacts in storage")
			return err
		}

		err = mc.FGetObject(context.Background(), bucket, object.Key, "/artifacts/incoming/", minio.GetObjectOptions{})
		if err != nil {
			logrus.WithError(err).Errorf("failed to download artifact %s", object.Key)
			//return err
		}
	}
	return nil
}

func ReadArtifactsFolder(dir string) []string {
	artifacts := []string{}
	files, err := ioutil.ReadDir("/artifacts/" + dir + "/")
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		if !f.IsDir() {
			artifacts = append(artifacts, f.Name())
		}
	}
	return artifacts
}

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

func jobSpec(name string, namespace string, image string, runid string, taskid string) *batchv1.Job {
	var ttl *int32
	ttl = new(int32)
	*ttl = 5
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-task-" + taskid,
			Namespace: namespace,
		},
		Spec: batchv1.JobSpec{
			//TTLSecondsAfterFinished: ttl,
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:            name,
							Image:           image,
							ImagePullPolicy: "Always",
							Command:         []string{"trinity"},
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

func minioPodSpec(name string, namespace string) *v1.Pod {
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
					Ports:   []v1.ContainerPort{{ContainerPort: 9000}},
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
