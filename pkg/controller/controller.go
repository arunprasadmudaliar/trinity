package controller

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
			//wf.name = utils.GetObjectMetaData(obj).Name
			wf.action = "create"
			//wf.namespace = utils.GetObjectMetaData(obj).Namespace
			if err == nil {
				q.Add(wf)
			}

			//logrus.Infof("Event received of type [%s] for [%s]", event.eventType, event.key)
		},
		UpdateFunc: func(old, new interface{}) {
			/* event.key, err = cache.MetaNamespaceKeyFunc(old)
			event.eventType = "update"
			if err == nil {
				q.Add(event)
			}
			logrus.Infof("Event received of type [%s] for [%s]", event.eventType, event.key) */
		},
		DeleteFunc: func(obj interface{}) {
			/* event.key, err = cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			event.eventType = "delete"
			if err == nil {
				q.Add(event)
			}
			logrus.Infof("Event received of type [%s] for [%s]", event.eventType, event.key) */
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
	e, term := c.queue.Get()

	if term {
		return false
	}

	err := c.processItem(e.(workflow))
	if err == nil {
		c.queue.Forget(e)
		return true
	}
	return true
}

func (c *controller) processItem(wf workflow) error {
	obj, _, err := c.informer.GetIndexer().GetByKey(wf.key)

	j, _ := obj.(*unstructured.Unstructured).MarshalJSON()
	var schedule schedule
	_ = json.Unmarshal(j, &schedule)

	if err != nil {
		return fmt.Errorf("Error fetching object with key %s from store: %v", wf.key, err)
	}

	ns := strings.Split(wf.key, "/")[0]
	name := strings.Split(wf.key, "/")[1]

	switch wf.action {
	case "create":
		err := utils.CreateCron(c.client.(*kubernetes.Clientset), name, ns, schedule.Spec.Schedule)
		if err != nil {
			logrus.WithError(err).Errorf("Failed to create workflow:%s", name)
			return err
		}
		logrus.Infof("Created cronjob %s scheduled to run at %s", name, schedule.Spec.Schedule)
	}

	//Use a switch clause instead and process the events based on the type

	return nil
}
