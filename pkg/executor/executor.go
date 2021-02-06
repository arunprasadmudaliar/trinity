package executor

import (
	"os/exec"

	wfv1 "github.com/arunprasadmudaliar/trinity/api/v1"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func Execute(config string, workflow string, namespace string, runid int, taskid int) {

	var cfg *rest.Config
	var err error
	if config == "" {
		cfg, err = rest.InClusterConfig()
		if err != nil {
			logrus.Fatalf("Error occured while reading incluster kubeconfig:%v", err)
			//return nil, err
		}
	} else {
		cfg, err = clientcmd.BuildConfigFromFlags("", config)
		if err != nil {
			logrus.WithError(err).Fatal("Failed to build workflow client configuration")
		}
	}

	kc, err := wfv1.NewForConfig(cfg)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create workflow client for given configuration")
	}

	//result := wfv1.Workflow{}
	wf, err := kc.WorkFlows(namespace).Get(workflow)
	if err != nil {
		logrus.WithError(err).Errorf("Failed to get workflow %s", workflow)
	}

	/* c := &exec.Cmd{
		//	Path: "/bin",
		//	Args: []string{"ls", "-a"},
		Path: wf.Spec.Tasks[taskid].Command,
		Args: wf.Spec.Tasks[taskid].Args,
	} */

	command := getCmd(wf.Spec.Tasks[taskid].Command)
	args := getArgs(wf.Spec.Tasks[taskid].Command, wf.Spec.Tasks[taskid].Args)
	c := exec.Command(command, args...)

	var st string
	wf.Kind = "Workflow"
	wf.APIVersion = "trinity.cloudlego.com/v1"

	var e string
	output, err := c.Output()
	if err != nil {
		st = "failed"
		e = err.Error()
	} else {
		st = "success"
		e = ""
	}

	taskstatus := wfv1.TaskStatus{
		Name:    wf.Spec.Tasks[taskid].Name,
		Command: command,
		Args:    args,
		Status:  st,
		Output:  string(output),
		Error:   e,
	}

	if taskid == len(wf.Spec.Tasks)-1 {
		wf.Status.Runs[runid].Phase = "completed"
	}

	wf.Status.Runs[runid].Tasks = append(wf.Status.Runs[runid].Tasks, taskstatus)

	_, err = kc.WorkFlows(namespace).Put(workflow, wf)
	if err != nil {
		logrus.WithError(err).Errorf("failed to update status for workflow %s in namespace %s", workflow, namespace)
	}

	logrus.Infof("updated status for workflow %s in namespace %s", workflow, namespace)

}
