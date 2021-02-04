package runner

import (
	"errors"
	"strconv"

	wfv1 "github.com/arunprasadmudaliar/trinity/api/v1"
	"github.com/arunprasadmudaliar/trinity/pkg/utils"
	batchv1 "k8s.io/api/batch/v1"
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
				ID:    1,
				Phase: "Running",
				Tasks: []wfv1.TaskStatus{},
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
		ID:    runid + 1,
		Phase: "running",
		Tasks: []wfv1.TaskStatus{},
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

func getImage(name string) string {
	switch name {
	case "shell":
		return "arunmudaliar/trinity:latest"
	default:
		return "arunmudaliar/trinity:latest"
	}
}

func deployJob(cfg string, name string, namespace string, workflow *wfv1.Workflow, runid string) {

	kc, err := utils.Client(cfg)
	if err != nil {
		logrus.Error(err)
	}

	for taskid, task := range workflow.Spec.Tasks {
		image := getImage((task.Type))
		_, err := utils.CreateJob(kc, name, namespace, image, runid, strconv.Itoa(taskid))

		if err != nil {
			logrus.Error(err)
		}

		logrus.Infof("executing task %s for workflow %s", task.Name, name)
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
						removeJob(kc, name+"-task-"+strconv.Itoa(taskid), namespace)
					} else {
						newerr := errors.New(object.Status.Conditions[0].Message)
						logrus.WithError(newerr).Errorf("failed to execute task %s", name)
						removeJob(kc, name+"-task-"+strconv.Itoa(taskid), namespace)
					}
					break
				} else {
					logrus.Errorf("failed to execute task %s", name)
					removeJob(kc, name+"-task-"+strconv.Itoa(taskid), namespace)
					break
				}

			} else if event.Type == watch.Added {
				logrus.Infof("waiting for task %s to complete", task.Name)
			}
		}
	}
}

func removeJob(kc *kubernetes.Clientset, name string, namespace string) {
	err := utils.DeleteJob(kc, name, namespace)
	if err != nil {
		logrus.WithError(err).Errorf("failed to remove task %s", name)
	} else {
		logrus.Infof("removed task %s", name)
	}
}
