package utils

import (
	"context"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"time"

	wfv1 "github.com/arunprasadmudaliar/trinity/api/v1"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func DeleteJobPod(kc *kubernetes.Clientset, name string, namespace string) error {
	pods, err := kc.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: "job-name=" + name,
	})

	if err != nil {
		return err
	}

	for _, pod := range pods.Items {
		err := kc.CoreV1().Pods(namespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func WatchPod(kc *kubernetes.Clientset, name string, namespace string) (watch.Interface, error) {
	opts := metav1.ListOptions{
		FieldSelector: "metadata.name=" + name,
	}
	return kc.CoreV1().Pods(namespace).Watch(context.Background(), opts)
}

func CreateJob(kc *kubernetes.Clientset, name string, namespace string, image string, runid string, taskid string, creds wfv1.MinioCreds) (*batchv1.Job, error) {
	jobspec := jobSpec(name, namespace, image, runid, taskid, creds)
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

func GetSecret(kc *kubernetes.Clientset, name string, namespace string) (map[string][]byte, v1.SecretType, error) {
	secret, err := kc.CoreV1().Secrets(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return nil, "", err
	}
	//logrus.Info(secret.StringData)

	return secret.Data, secret.Type, nil
}

func DeployMinio(kc *kubernetes.Clientset, name string, namespace string, creds wfv1.MinioCreds) (*v1.Pod, *v1.Service, error) {
	podspec := minioPodSpec(name, namespace, creds)
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

func MinioCredential() string {
	rand.Seed(time.Now().UnixNano())

	letterRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_@#&12345")
	b := make([]rune, 20)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func UploadArtifacts(bucket string, url string, artifacts []string) error {

	ctx := context.Background()
	endpoint := url
	accessKeyID := os.Getenv("MINIO_ROOT_USER")
	secretAccessKey := os.Getenv("MINIO_ROOT_PASSWORD")
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
			return err
		}

	}

	return nil
}

func DownloadArtifacts(bucket string, url string) error {
	endpoint := url
	accessKeyID := os.Getenv("MINIO_ROOT_USER")
	secretAccessKey := os.Getenv("MINIO_ROOT_PASSWORD")
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

		err = mc.FGetObject(context.Background(), bucket, object.Key, "/artifacts/incoming/"+object.Key, minio.GetObjectOptions{})
		if err != nil {
			logrus.WithError(err).Errorf("failed to download artifact %s", object.Key)
			return err
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

func Timestamp() string {
	dt := time.Now()
	return dt.Format("01-02-2006 15:04:05")
}
