package executor

import (
	"io/ioutil"
	"os"
	"os/exec"

	wfv1 "github.com/arunprasadmudaliar/trinity/api/v1"
	"github.com/arunprasadmudaliar/trinity/pkg/utils"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type phase struct {
	kc       *kubernetes.Clientset
	wf       *wfv1.WorkFlowClient
	workflow *wfv1.Workflow
	runid    int
	taskid   int
}

func Execute(config string, workflow string, namespace string, runid int, taskid int) {

	var cfg *rest.Config
	var err error
	storageendpoint := workflow + "-artifact-svc." + namespace + ".svc.cluster.local"

	kc, err := utils.Client(config)
	if err != nil {
		logrus.WithError(err).Fatal("failed to create default client for given configuration")
	}

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

	wc, err := wfv1.NewForConfig(cfg)
	if err != nil {
		logrus.WithError(err).Fatal("failed to create workflow client for given configuration")
	}

	//result := wfv1.Workflow{}
	wf, err := wc.WorkFlows(namespace).Get(workflow)
	if err != nil {
		logrus.WithError(err).Errorf("failed to get workflow %s", workflow)
	}

	phase := phase{
		kc:       kc,
		wf:       wc,
		workflow: wf,
		runid:    runid,
		taskid:   taskid,
	}

	//Inject Input Vars
	phase.injectInputVars()

	//Download artifacts
	phase.downloadArtifacts(storageendpoint)

	//Execute commands
	output, err := phase.execute()

	//upload artifacts
	phase.uploadArtifacts(storageendpoint)

	//update status
	phase.updateStatus(output, err)

}

func setEnvVar(envvar string) error {
	return os.Setenv("WF_INPUT", envvar)
}

func execScript(script string) ([]byte, error) {
	err := ioutil.WriteFile("./workflow.sh", []byte(script), 0777)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command("./workflow.sh")
	return cmd.Output()
}

func (p phase) injectInputVars() {
	//Inject input variable
	if p.taskid > 0 {
		err := setEnvVar(p.workflow.Status.Runs[p.runid].Tasks[p.taskid-1].Output)
		if err != nil {
			logrus.WithError(err).Errorf("failed to inject input for task %d", p.taskid)
		}
	} else {
		logrus.Info("no need to inject input since this is first task")
	}
}

func (p phase) injectSecrets() {
	secrets := p.workflow.Spec.Tasks[p.taskid].Secrets
	for _, secret := range secrets {
		data, _ := utils.GetSecret(p.kc, secret, p.workflow.Namespace)
	}
}

func (p phase) downloadArtifacts(storageendpoint string) {
	//Check if artifact store is used.If yes, download artifacts
	if os.Getenv("MINIO_ROOT_USER") != "" {
		if p.taskid > 0 {
			err := utils.DownloadArtifacts(p.workflow.Name, storageendpoint)
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
}

func (p phase) execute() ([]byte, error) {
	var output []byte
	var err error

	if p.workflow.Spec.Tasks[p.taskid].Command.Script != "" {
		output, err = execScript(p.workflow.Spec.Tasks[p.taskid].Command.Script)
	} else {
		output, err = exec.Command(p.workflow.Spec.Tasks[p.taskid].Command.Inline.Command, p.workflow.Spec.Tasks[p.taskid].Command.Inline.Args...).Output()
	}
	return output, err
}

func (p phase) uploadArtifacts(storageendpoint string) {
	//upload artifacts if artifact store is enabled. Skip for the last task.
	if os.Getenv("MINIO_ROOT_USER") != "" {
		if p.taskid != len(p.workflow.Spec.Tasks)-1 {
			artifacts := utils.ReadArtifactsFolder("outgoing")
			if len(artifacts) > 0 {

				err := utils.UploadArtifacts(p.workflow.Name, storageendpoint, artifacts)
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
}

func (p phase) updateStatus(output []byte, err error) {
	var st string
	p.workflow.Kind = "Workflow"
	p.workflow.APIVersion = "trinity.cloudlego.com/v1"

	var e string
	//output, err := c.Output()
	if err != nil {
		st = "failed"
		e = err.Error()
	} else {
		st = "success"
		e = ""
	}
	taskstatus := wfv1.TaskStatus{
		Name:   p.workflow.Spec.Tasks[p.taskid].Name,
		Status: st,
		Output: string(output),
		Error:  e,
	}

	if p.taskid == len(p.workflow.Spec.Tasks)-1 {
		p.workflow.Status.Runs[p.runid].Phase = "completed"
		p.workflow.Status.Runs[p.runid].EndedAt = utils.Timestamp()
	}

	p.workflow.Status.Runs[p.runid].Tasks = append(p.workflow.Status.Runs[p.runid].Tasks, taskstatus)

	_, err = p.wf.WorkFlows(p.workflow.Namespace).Put(p.workflow.Name, p.workflow)
	if err != nil {
		logrus.WithError(err).Errorf("failed to update status for workflow %s in namespace %s", p.workflow.Name, p.workflow.Namespace)
	}

	logrus.Infof("updated status for workflow %s in namespace %s", p.workflow.Name, p.workflow.Namespace)

}
