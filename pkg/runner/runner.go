package runner

import (
	"errors"
	"strconv"

	wfv1 "github.com/arunprasadmudaliar/trinity/api/v1"
	"github.com/arunprasadmudaliar/trinity/pkg/utils"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/sirupsen/logrus"
)

//Run will trigger the executor
func Run(config string, name string, ns string) {
	cfg, err := clientcmd.BuildConfigFromFlags("", config)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to build workflow client configuration")
	}

	kc, err := wfv1.NewForConfig(cfg)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create workflow client for given configuration")
	}

	workflow, err := kc.WorkFlows(ns).Get(name)
	if err != nil {
		logrus.Error(err)
	}

	var runid int
	if len(workflow.Status.Runs) == 0 {
		runid, _ = initialRun(kc, name, ns, workflow)
	} else {
		runid, _ = nextRun(kc, name, ns, workflow)
	}
	deployJob(config, name, ns, workflow, strconv.Itoa(runid))
}

func initialRun(kc *wfv1.WorkFlowClient, name string, namespace string, workflow *wfv1.Workflow) (int, error) {
	init := wfv1.WorkflowStatus{
		Runs: []wfv1.Workflowruns{
			{
				ID:        1,
				Phase:     "Running",
				Tasks:     []wfv1.TaskStatus{},
				StartedAt: utils.Timestamp(),
				EndedAt:   "",
			},
		},
	}

	workflow.Status = init
	workflow.Kind = "Workflow"
	workflow.APIVersion = "trinity.cloudlego.com/v1"
	_, err := kc.WorkFlows(namespace).Put(name, workflow)

	if err != nil {
		return -1, err
	}
	logrus.Infof("triggered run %d for workflow %s under namespace %s", 1, name, namespace)
	return 0, nil
}

func nextRun(kc *wfv1.WorkFlowClient, name string, namespace string, workflow *wfv1.Workflow) (int, error) {

	runid := len(workflow.Status.Runs)

	runstatus := wfv1.Workflowruns{
		ID:        runid + 1,
		Phase:     "running",
		Tasks:     []wfv1.TaskStatus{},
		StartedAt: utils.Timestamp(),
		EndedAt:   "",
	}

	workflow.Status.Runs = append(workflow.Status.Runs, runstatus)
	workflow.Kind = "Workflow"
	workflow.APIVersion = "trinity.cloudlego.com/v1"
	_, err := kc.WorkFlows(namespace).Put(name, workflow)

	if err != nil {
		return -1, err
	}
	logrus.Infof("triggered run %d for workflow %s under namespace %s", runid+1, name, namespace)
	return runid, nil
}

func deployJob(cfg string, name string, namespace string, workflow *wfv1.Workflow, runid string) {

	kc, err := utils.Client(cfg)
	if err != nil {
		logrus.Error(err)
	}

	var minio *v1.Pod
	var svc *v1.Service
	var artifactEnabled bool
	var creds wfv1.MinioCreds
	//deploy minio to store artifacts
	if workflow.Spec.StoreArtifacts {
		artifactEnabled = true

		creds = wfv1.MinioCreds{
			AccessKey: utils.MinioCredential(),
			SecretKey: utils.MinioCredential(),
		}

		minio, svc, err = utils.DeployMinio(kc, name, namespace, creds)
		if err != nil {
			logrus.WithError(err).Errorf("failed to initialize artifact store")
		}

		logrus.Info("artifact store is up and running")
	}

	for taskid, task := range workflow.Spec.Tasks {

		//image := getImage((task.Command))
		job, err := utils.CreateJob(kc, name, namespace, IMAGE, runid, strconv.Itoa(taskid), creds)

		if err != nil {
			logrus.Error(err)
		}

		logrus.Infof("executing task %s for workflow %s", task.Name, name)
		//ch, err := utils.WatchJob(kc, name+"-task-"+strconv.Itoa(taskid), namespace)
		ch, err := utils.WatchJob(kc, name+"-task-"+strconv.Itoa(taskid), namespace)
		if err != nil {
			logrus.Error(err)
		}

		for event := range ch.ResultChan() {
			if event.Type == watch.Modified {
				object := event.Object.(*batchv1.Job)

				if len(object.Status.Conditions) > 0 {
					if object.Status.Conditions[0].Type == "Complete" {
						logrus.Infof("completed task %s for workflow %s", task.Name, name)
						//removeJob(kc, name+"-task-"+strconv.Itoa(taskid), namespace)
						removeJob(kc, job)
					} else {
						newerr := errors.New(object.Status.Conditions[0].Message)
						logrus.WithError(newerr).Errorf("failed to execute task %s", name)
						//removeJob(kc, name+"-task-"+strconv.Itoa(taskid), namespace)
						removeJob(kc, job)
					}
					break
				}
			} else if event.Type == watch.Added {
				logrus.Infof("waiting for task %s to complete", task.Name)
			}
		}
	}

	//Perform cleanup of artifactory storage
	if artifactEnabled {
		err := utils.DeleteMinio(kc, minio, svc)
		if err != nil {
			logrus.WithError(err).Errorf("failed to delete artifact store")
		}
		logrus.Info("artifact store was removed successfully")
	}
}

//func removeJob(kc *kubernetes.Clientset, name string, namespace string) {
func removeJob(kc *kubernetes.Clientset, job *batchv1.Job) {
	err := utils.DeleteJob(kc, job.ObjectMeta.Name, job.ObjectMeta.Namespace)
	if err != nil {
		logrus.WithError(err).Errorf("failed to remove job %s.Manual clean up required before next run.", job.ObjectMeta.Name)
	}
	err = utils.DeleteJobPod(kc, job.ObjectMeta.Name, job.ObjectMeta.Namespace)
	if err != nil {
		logrus.WithError(err).Errorf("failed to remove pod for job %s.Manual clean up required before next run.", job.ObjectMeta.Name)
	}

	logrus.Info("jobs and corresponding pods were removed successfully")
}
