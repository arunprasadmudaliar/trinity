package executor

import (
	"context"

	wfv1 "github.com/arunprasadmudaliar/trinity/api/v1"

	"github.com/arunprasadmudaliar/trinity/pkg/utils"
	"github.com/sirupsen/logrus"
)

//Run will trigger the executor
func Run(config string) {
	kc, err := utils.WfClient(config)
	if err != nil {
		logrus.Fatal(err)
	}

	result := wfv1.WorkflowList{}
	err = kc.Get().Namespace("default").Resource("workflows").Do(context.Background()).Into(&result)
	if err != nil {
		logrus.Error(err)
	}

	logrus.Info(result)
}
