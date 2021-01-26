package controller

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	wfv1 "github.com/arunprasadmudaliar/trinity/api/v1"
	"github.com/arunprasadmudaliar/trinity/pkg/utils"
	"github.com/sirupsen/logrus"

	v1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/kubernetes"

	//"k8s.io/client-go/informers"

	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
)

const maxRetries = 5

type workflow struct {
	key string
	//name      string
	action string
	//namespace string
	schedule string
}

type controller struct {
	client   kubernetes.Interface
	informer cache.SharedIndexInformer
	queue    workqueue.RateLimitingInterface
}

type schedule struct {
	Spec struct {
		Schedule string
	}
}

//Start fuction with start the controller
func Start(config string) {
	// This client will be used when a clientset is required
	kc, err := utils.Client(config)
	if err != nil {
		logrus.Fatal(err)
	}

	//Below client is created directly from config without a clientset
	cfg, err := clientcmd.BuildConfigFromFlags("", config)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to build client configuration")
	}

	dc, err := dynamic.NewForConfig(cfg)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create dynamic client for given configuration")
	}

	f := dynamicinformer.NewFilteredDynamicSharedInformerFactory(dc, 0, v1.NamespaceAll, nil)
	gvr, _ := schema.ParseResourceArg("workflows.v1.trinity.cloudlego.com") // group.version.resource or GroupVersionResource

	i := f.ForResource(*gvr) // we create a new informer here

	//factory := informers.NewSharedInformerFactory(kc, 0) // For clientset
	//informer := factory.Core().V1().Pods().Informer()
	//c := newController(kc, i.Informer())

	c := newController(kc, i.Informer())
	stopCh := make(chan struct{})
	defer close(stopCh)

	c.Run(stopCh)

}

func newController(kc *kubernetes.Clientset, informer cache.SharedIndexInformer) *controller {
	q := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	var wf workflow
	var err error
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			wf.key, err = cache.MetaNamespaceKeyFunc(obj)
			wf.action = "create"

			if err == nil {
				q.Add(wf)
			}
		},
		UpdateFunc: func(old, new interface{}) {
			wf.key, err = cache.MetaNamespaceKeyFunc(old)
			wf.action = "update"
			if err == nil {
				q.Add(wf)
			}
		},
		DeleteFunc: func(obj interface{}) {
			wf.key, err = cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			wf.action = "delete"
			if err == nil {
				q.Add(wf)
			}

		},
	})
	return &controller{
		client:   kc,
		informer: informer,
		queue:    q,
	}
}

func (c *controller) Run(stopper <-chan struct{}) {

	defer utilruntime.HandleCrash() //this will handle panic and won't crash the process
	defer c.queue.ShutDown()        //shutdown all workqueue and terminate all workers

	logrus.Info("Starting workflow controller...")

	go c.informer.Run(stopper)

	logrus.Info("Synchronizing events...")

	//synchronize the cache before starting to process events
	if !cache.WaitForCacheSync(stopper, c.informer.HasSynced) {
		utilruntime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		logrus.Info("synchronization failed...")
		return
	}

	logrus.Info("synchronization complete!")
	//logrus.Info("")

	wait.Until(c.runWorker, time.Second, stopper)
}

func (c *controller) runWorker() {
	for c.processNextItem() {
		// continue looping
	}
}

func (c *controller) processNextItem() bool {
	wf, quit := c.queue.Get()

	if quit {
		return false
	}
	defer c.queue.Done(wf.(workflow))
	err := c.processItem(wf.(workflow))
	if err == nil {
		// No error, reset the ratelimit counters
		c.queue.Forget(wf)
	} /* else if c.queue.NumRequeues(wf) < maxRetries {
		logrus.Errorf("Error processing %s (will retry): %v", wf.(workflow).key, err)
		c.queue.AddRateLimited(wf)
	} else {
		// err != nil and too many retries
		logrus.Errorf("Error processing %s (giving up): %v", wf.(workflow).key, err)
		c.queue.Forget(wf)
		utilruntime.HandleError(err)
	} */

	return true
}

func (c *controller) processItem(wf workflow) error {
	obj, _, err := c.informer.GetIndexer().GetByKey(wf.key)
	if err != nil {
		logrus.WithError(err).Errorf("Failed to fetch workflow %s from store", wf.key)
		return err
	}

	var schedule wfv1.Workflow
	if wf.action != "delete" {
		j, err := obj.(*unstructured.Unstructured).MarshalJSON()
		if err != nil {
			logrus.WithError(err).Errorf("Failed to process %s event for workflow %s", wf.action, wf.key)
			return err
		}
		err = json.Unmarshal(j, &schedule)
		if err != nil {
			logrus.WithError(err).Errorf("Failed to process %s event for workflow %s", wf.action, wf.key)
			return err
		}
	}

	ns := strings.Split(wf.key, "/")[0]
	name := strings.Split(wf.key, "/")[1]

	switch wf.action {
	case "create":
		created, err := utils.CreateCron(c.client.(*kubernetes.Clientset), name, ns, schedule.Spec.Schedule)
		if err != nil {
			logrus.WithError(err).Errorf("Failed to create Cron for Workflow:%s", name)
			return err
		}

		if created {
			logrus.Infof("Created Cron wf-cron-%s for Workflow %s", name, name)
			return nil
		}
		logrus.Infof("Found a Cron wf-cron-%s for Workflow %s", name, name)
		return nil

	case "update":
		err = utils.UpdateCron(c.client.(*kubernetes.Clientset), name, ns, schedule.Spec.Schedule)
		if err != nil {
			logrus.WithError(err).Errorf("Failed to Update Cron wf-cron-%s for Workflow %s", name, name)
			return err
		}
		logrus.Infof("Updated Cron wf-cron-%s for Workflow %s.Scheduled time %s", name, name, schedule.Spec.Schedule)

	case "delete":
		deleted, err := utils.DeleteCron(c.client.(*kubernetes.Clientset), name, ns)
		if err != nil {
			logrus.WithError(err).Errorf("Failed to delete Cron wf-cron-%s for Workflow:%s", name, name)
			return err
		}
		if deleted {
			logrus.Infof("Removed Cron wf-cron-%s for workflow %s", name, name)
			return nil
		}

		logrus.Infof("Did not find a Cron wf-cron-%s for workflow %s to delete", name, name)
		return nil
	}
	return nil

}
