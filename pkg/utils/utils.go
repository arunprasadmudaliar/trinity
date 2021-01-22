package utils

import (
	"context"

	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	batch "k8s.io/api/batch/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func cronJobSpec(name string, namespace string, schedule string) *batch.CronJob {
	return &batch.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "wf-cron-" + name,
			Namespace: namespace,
		},
		Spec: batch.CronJobSpec{
			Schedule: schedule,
			JobTemplate: batch.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name:            name,
									Image:           "busybox",
									ImagePullPolicy: v1.PullIfNotPresent,
									Command: []string{
										"sleep",
										"10",
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