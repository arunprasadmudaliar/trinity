package executor

import (
	"io/ioutil"
	"os"
	"os/exec"

	wfv1 "github.com/arunprasadmudaliar/trinity/api/v1"
	"github.com/arunprasadmudaliar/trinity/pkg/utils"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func Execute(config string, workflow string, namespace string, runid int, taskid int) {

	var cfg *rest.Config
	var err error
	storageendpoint := workflow + "-artifact-svc." + namespace + ".svc.cluster.local"

	if config == "" {
		cfg, err = rest.InClusterConfig()
		if err != nil {
			logrus.Fatalf("error occured while reading incluster kubeconfig:%v", err)
			//return nil, err
		}
	} else {
		cfg, err = clientcmd.BuildConfigFromFlags("", config)
		if err != nil {
			logrus.WithError(err).Fatal("failed to build workflow client configuration")
		}
	}

	kc, err := wfv1.NewForConfig(cfg)
	if err != nil {
		logrus.WithError(err).Fatal("failed to create workflow client for given configuration")
	}

	//result := wfv1.Workflow{}
	wf, err := kc.WorkFlows(namespace).Get(workflow)
	if err != nil {
		logrus.WithError(err).Errorf("failed to get workflow %s", workflow)
	}

	//Inject input variable
	if taskid > 0 {
		err := inputVar(wf.Status.Runs[runid].Tasks[taskid-1].Output)
		if err != nil {
			logrus.WithError(err).Errorf("failed to inject input for task %d", taskid)
		}
	} else {
		logrus.Info("no need to inject input since this is first task")
	}

	//Check if artifact store is used.If yes, download artifacts
	if os.Getenv("MINIO_ROOT_USER") != "" {
		if taskid > 0 {
			err = utils.DownloadArtifacts(workflow, storageendpoint)
			if err != nil {
				logrus.WithError(err).Info("failed to download artifacts")
			}
			logrus.Info("artifacts were downloaded successfully")
		} else {
			logrus.Info("no need to download artifacts since this is the first task")
		}
	} else {
		logrus.Info("skipping artifact download since artifact store is not used")
	}

	var output []byte

	if wf.Spec.Tasks[taskid].Command.Script != "" {
		output, err = execScript(wf.Spec.Tasks[taskid].Command.Script)
	} else {
		output, err = exec.Command(wf.Spec.Tasks[taskid].Command.Inline.Command, wf.Spec.Tasks[taskid].Command.Inline.Args...).Output()
	}

	//command := getCmd(wf.Spec.Tasks[taskid].Command)
	//args := getArgs(wf.Spec.Tasks[taskid].Command, wf.Spec.Tasks[taskid].Args)
	//c := exec.Command(command, args...)

	var st string
	wf.Kind = "Workflow"
	wf.APIVersion = "trinity.cloudlego.com/v1"

	var e string
	//output, err := c.Output()
	if err != nil {
		st = "failed"
		e = err.Error()
	} else {
		st = "success"
		e = ""
	}

	//upload artifacts if artifact store is enabled. Skip for the last task.
	if os.Getenv("MINIO_ROOT_USER") != "" {
		if taskid != len(wf.Spec.Tasks)-1 {
			artifacts := utils.ReadArtifactsFolder("outgoing")
			if len(artifacts) > 0 {

				err := utils.UploadArtifacts(workflow, storageendpoint, artifacts)
				if err != nil {
					logrus.WithError(err).Errorf("failed to upload artifacts")
				}
				logrus.Info("artifacts were uploaded successfully")
			} else {
				logrus.Info("no artifacts to upload")
			}
		} else {
			logrus.Info("skipping artifacts upload since this is the last task")
		}
	} else {
		logrus.Info("skipping artifact upload since artifact store is not used")
	}

	taskstatus := wfv1.TaskStatus{
		Name:   wf.Spec.Tasks[taskid].Name,
		Status: st,
		Output: string(output),
		Error:  e,
	}

	if taskid == len(wf.Spec.Tasks)-1 {
		wf.Status.Runs[runid].Phase = "completed"
		wf.Status.Runs[runid].EndedAt = utils.Timestamp()
	}

	wf.Status.Runs[runid].Tasks = append(wf.Status.Runs[runid].Tasks, taskstatus)

	_, err = kc.WorkFlows(namespace).Put(workflow, wf)
	if err != nil {
		logrus.WithError(err).Errorf("failed to update status for workflow %s in namespace %s", workflow, namespace)
	}

	logrus.Infof("updated status for workflow %s in namespace %s", workflow, namespace)

}

func inputVar(input string) error {
	return os.Setenv("WF_INPUT", input)
}

func execScript(script string) ([]byte, error) {
	err := ioutil.WriteFile("./workflow.sh", []byte(script), 0777)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command("./workflow.sh")
	return cmd.Output()
}
