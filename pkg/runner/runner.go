package runner

import (
	wfv1 "github.com/arunprasadmudaliar/trinity/api/v1"
	"github.com/arunprasadmudaliar/trinity/pkg/utils"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
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

	deployPod(config, name, ns, workflow)

	if len(workflow.Status.Runs) == 0 {
		initialRun(kc, name, ns, workflow)
	} else {
		nextRun(kc, name, ns, workflow)
	}

}

func initialRun(kc *wfv1.WorkFlowClient, name string, namespace string, workflow *wfv1.Workflow) error {
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
		return err
	}
	logrus.Infof("triggered run %d for workflow %s under namespace %s", 1, name, namespace)
	return nil
}

func nextRun(kc *wfv1.WorkFlowClient, name string, namespace string, workflow *wfv1.Workflow) error {

	runid := len(workflow.Status.Runs) + 1

	runstatus := wfv1.Workflowruns{
		ID:    runid,
		Phase: "running",
		Tasks: []wfv1.TaskStatus{},
	}

	workflow.Status.Runs = append(workflow.Status.Runs, runstatus)
	workflow.Kind = "Workflow"
	workflow.APIVersion = "trinity.cloudlego.com/v1"
	_, err := kc.WorkFlows(namespace).Put(name, workflow)

	if err != nil {
		return err
	}
	logrus.Infof("triggered run %d for workflow %s under namespace %s", runid, name, namespace)
	return nil
}

func getImage(name string) string {
	switch name {
	case "shell":
		return "alpine:latest"
	default:
		return "alpine:latest"
	}
}

func deployPod(cfg string, name string, namespace string, workflow *wfv1.Workflow) {

	kc, err := utils.Client(cfg)
	if err != nil {
		logrus.Error(err)
	}

	for _, task := range workflow.Spec.Tasks {
		image := getImage((task.Type))
		_, err := utils.CreatePod(kc, name, namespace, image)

		if err != nil {
			logrus.Error(err)
		}
		logrus.Infof("executing task %s for workflow %s", task.Name, name)
		ch, err := utils.WatchPod(kc, name, namespace)
		if err != nil {
			logrus.Error(err)
		}

		for event := range ch.ResultChan() {
			if event.Type == watch.Modified {
				podobject := event.Object.(*v1.Pod)

				// logrus.Info(podobject.Status.ContainerStatuses[0].State)
				state := podobject.Status.ContainerStatuses[0].State
				if state.Terminated != nil && state.Terminated.Reason == "Completed" {
					logrus.Infof("completed task %s for workflow %s", task.Name, name)
					break
				}
				logrus.Infof("waiting for task %s to complete", task.Name)
			}
		}
	}
}
