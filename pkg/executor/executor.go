package executor

import (
	"os/exec"

	wfv1 "github.com/arunprasadmudaliar/trinity/api/v1"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/clientcmd"
)

func Execute(config string, workflow string, namespace string, runid int, taskid int) {

	cfg, err := clientcmd.BuildConfigFromFlags("", config)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to build workflow client configuration")
	}

	kc, err := wfv1.NewForConfig(cfg)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create workflow client for given configuration")
	}

	//result := wfv1.Workflow{}
	wf, err := kc.WorkFlows(namespace).Get(workflow)
	if err != nil {
		logrus.Error(err)
	}

	c := &exec.Cmd{
		Path: wf.Spec.Tasks[taskid].Command,
		Args: wf.Spec.Tasks[taskid].Args,
	}

	var st string
	wf.Kind = "Workflow"
	wf.APIVersion = "trinity.cloudlego.com/v1"

	output, err := c.Output()
	if err != nil {
		st = "failed"
	} else {
		st = "success"
	}

	taskstatus := wfv1.TaskStatus{
		Name:   wf.Spec.Tasks[taskid].Name,
		Type:   wf.Spec.Tasks[taskid].Type,
		Status: st,
		Output: string(output),
		Error:  err.Error(),
	}

	if taskid == len(wf.Spec.Tasks)-1 {
		wf.Status.Runs[runid].Phase = "completed"
	}

	wf.Status.Runs[runid].Tasks = append(wf.Status.Runs[runid].Tasks, taskstatus)

	_, err = kc.WorkFlows(namespace).Put(workflow, wf)
	if err != nil {
		logrus.WithError(err).Errorf("failed to update prerunstatus for workflow %s in namespace %s", workflow, namespace)
	}

	logrus.Infof("updated prerunstatus for workflow %s in namespace %s", workflow, namespace)

}
