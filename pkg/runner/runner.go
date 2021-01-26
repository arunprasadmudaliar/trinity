package runner

import (
	wfv1 "github.com/arunprasadmudaliar/trinity/api/v1"
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

	//result := wfv1.Workflow{}
	result, err := kc.WorkFlows(ns).Get(name)
	//result, err := kc.WorkFlows(ns).List()
	//err = kc.Get().Namespace(ns).Name(name).Resource("workflows").Do(context.Background()).Into(&result)
	if err != nil {
		logrus.Error(err)
	}

	logrus.Info(result.Spec.Tasks)
}
