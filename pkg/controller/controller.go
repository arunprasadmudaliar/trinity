package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/arunprasadmudaliar/trinity/pkg/utils"
	"github.com/sirupsen/logrus"

	batchv1 "k8s.io/api/batch/v1"
	batch "k8s.io/api/batch/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

type event struct {
	key          string
	eventType    string
	resourceType string
}

type controller struct {
	client   kubernetes.Interface
	informer cache.SharedIndexInformer
	queue    workqueue.RateLimitingInterface
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
	var event event
	var err error
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			event.key, err = cache.MetaNamespaceKeyFunc(obj)
			event.eventType = "create"
			if err == nil {
				q.Add(event)
			}

			cj, err := kc.BatchV1beta1().CronJobs("default").Create(context.Background(), cronJob(), metav1.CreateOptions{})
			if err != nil {
				logrus.WithError(err).Fatal("Failed to create cronjob")
			}
			logrus.Info(cj)

			logrus.Infof("Event received of type [%s] for [%s]", event.eventType, event.key)
		},
		UpdateFunc: func(old, new interface{}) {
			event.key, err = cache.MetaNamespaceKeyFunc(old)
			event.eventType = "update"
			if err == nil {
				q.Add(event)
			}
			logrus.Infof("Event received of type [%s] for [%s]", event.eventType, event.key)
		},
		DeleteFunc: func(obj interface{}) {
			event.key, err = cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			event.eventType = "delete"
			if err == nil {
				q.Add(event)
			}
			logrus.Infof("Event received of type [%s] for [%s]", event.eventType, event.key)
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

	logrus.Info("Starting Chronos...")

	go c.informer.Run(stopper)

	logrus.Info("Synchronizing events...")

	//synchronize the cache before starting to process events
	if !cache.WaitForCacheSync(stopper, c.informer.HasSynced) {
		utilruntime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		logrus.Info("synchronization failed...")
		return
	}

	logrus.Info("synchronization complete!")
	logrus.Info("Ready to process events")

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

	err := c.processItem(e.(event))
	if err == nil {
		c.queue.Forget(e)
		return true
	}
	return true
}

func (c *controller) processItem(e event) error {
	obj, _, err := c.informer.GetIndexer().GetByKey(e.key)
	if err != nil {
		return fmt.Errorf("Error fetching object with key %s from store: %v", e.key, err)
	}

	//Use a switch clause instead and process the events based on the type
	logrus.Infof("Chronos has processed 1 event of type [%s] for object [%s]", e.eventType, obj)

	return nil
}

func cronJob() *batch.CronJob {

	return &batch.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mycron",
			Namespace: "default",
		},
		Spec: batch.CronJobSpec{
			Schedule: "*/1 * * * *",
			JobTemplate: batch.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name:            "busybox",
									Image:           "busybox",
									ImagePullPolicy: v1.PullIfNotPresent,
									Command: []string{
										"sleep",
										"10",
									},
								},
							},
							RestartPolicy: "Never",
						},
					},
				},
			},
		},
	}
}
