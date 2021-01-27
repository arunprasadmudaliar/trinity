package executor

import (
	"os/exec"

	wfv1 "github.com/arunprasadmudaliar/trinity/api/v1"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/clientcmd"
)

type status struct {
	kc        *wfv1.WorkFlowClient
	workflow  string
	wf        *wfv1.Workflow
	namespace string
	task      string
	taskid    int
}

func Execute(config string, workflow string, namespace string, id int) {
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

	update := status{
		kc:        kc,
		workflow:  workflow,
		namespace: namespace,
		wf:        wf,
		taskid:    id,
	}
	update.prerunstatus()

	c := &exec.Cmd{
		Path: wf.Spec.Tasks[id].Command,
		Args: wf.Spec.Tasks[id].Args,
	}

	output, err := c.Output()

	update.postrunstatus(string(output), err)
}

func (s status) prerunstatus() {
	runid := len(s.wf.Status.Runs)
	logrus.Info(runid)
	s.wf.Status.Runs[runid].Tasks[s.taskid].Name = s.wf.Spec.Tasks[s.taskid].Name
	s.wf.Status.Runs[runid].Tasks[s.taskid].Type = s.wf.Spec.Tasks[s.taskid].Type
	s.wf.Status.Runs[runid].Tasks[s.taskid].Status = "running"

	_, err := s.kc.WorkFlows(s.namespace).Put(s.workflow, s.wf)
	if err != nil {
		logrus.WithError(err).Errorf("failed to update prerunstatus for workflow %s in namespace %s", s.workflow, s.namespace)
	}

	logrus.Infof("updated prerunstatus for workflow %s in namespace %s", s.workflow, s.namespace)

}

func (s status) postrunstatus(output string, err error) {
	runid := len(s.wf.Status.Runs)
	if err != nil {
		s.wf.Status.Runs[runid].Tasks[s.taskid].Output = output
		s.wf.Status.Runs[runid].Tasks[s.taskid].Status = "failed"
		s.wf.Status.Runs[runid].Tasks[s.taskid].Error = err.Error()
	}

	s.wf.Status.Runs[runid].Tasks[s.taskid].Output = output
	s.wf.Status.Runs[runid].Tasks[s.taskid].Status = "success"

	_, err = s.kc.WorkFlows(s.namespace).Put(s.workflow, s.wf)
	if err != nil {
		logrus.WithError(err).Errorf("failed to update postrunstatus for workflow %s in namespace %s", s.workflow, s.namespace)
	}

	logrus.Infof("updated postrunstatus for workflow %s in namespace %s", s.workflow, s.namespace)
}
