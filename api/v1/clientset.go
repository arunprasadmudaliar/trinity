package v1

import (
	"context"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

type WorkFlowV1Interface interface {
	WorkFlows(namespace string) WorkFlowInterface
}

type WorkFlowClient struct {
	restClient rest.Interface
}

type WorkFlowInterface interface {
	List() (*WorkflowList, error)
	Get(name string) (*Workflow, error)
	Put(name string, workflow *Workflow) (*Workflow, error)
	//Create(*v1alpha1.Project) (*v1alpha1.Project, error)
	//Watch(opts metav1.ListOptions) (watch.Interface, error)
}

type workflowclient struct {
	restClient rest.Interface
	ns         string
}

func NewForConfig(config *rest.Config) (*WorkFlowClient, error) {
	crdConfig := *config
	crdConfig.ContentConfig.GroupVersion = &schema.GroupVersion{Group: "trinity.cloudlego.com", Version: "v1"}
	crdConfig.APIPath = "/apis"
	crdConfig.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	crdConfig.UserAgent = rest.DefaultKubernetesUserAgent()

	client, err := rest.RESTClientFor(&crdConfig)
	if err != nil {
		logrus.Fatalf("Error occured while creating workflow client:%v", err)
	}

	return &WorkFlowClient{restClient: client}, nil
}

/* func WfClient(configpath string) (*rest.RESTClient, error) {

	var config *rest.Config
	var err error

	if configpath == "" {
		logrus.Info("Using Incluster configuration")
		config, err = rest.InClusterConfig()
	} else {
		logrus.Infof("Using configuration file:%s", configpath)
		config, err = clientcmd.BuildConfigFromFlags("", configpath)
	}

	if err != nil {
		logrus.Fatalf("Error occured while reading kubeconfig:%v", err)
		return nil, err
	}

	//	wfv1.AddToScheme(scheme.Scheme)

	crdConfig := *config
	crdConfig.ContentConfig.GroupVersion = &schema.GroupVersion{Group: "trinity.cloudlego.com", Version: "v1"}
	crdConfig.APIPath = "/apis"
	crdConfig.NegotiatedSerializer = serializer.NewCodecFactory(scheme.Scheme)
	crdConfig.UserAgent = rest.DefaultKubernetesUserAgent()

	exampleRestClient, err := rest.RESTClientFor(&crdConfig)
	if err != nil {
		logrus.Fatalf("Error occured while creating workflow client:%v", err)
	}

	return exampleRestClient, nil
} */

func (c *WorkFlowClient) WorkFlows(namespace string) WorkFlowInterface {
	return &workflowclient{
		restClient: c.restClient,
		ns:         namespace,
	}
}

func (c *workflowclient) List() (*WorkflowList, error) {
	result := WorkflowList{}
	err := c.restClient.
		Get().
		Namespace(c.ns).
		Resource("workflows").
		//VersionedParams(&opts, scheme.ParameterCodec).
		Do(context.Background()).
		Into(&result)

	return &result, err
}

func (c *workflowclient) Get(name string) (*Workflow, error) {
	result := Workflow{}
	err := c.restClient.
		Get().
		Namespace(c.ns).
		Resource("workflows").
		Name(name).
		SubResource("status").
		//VersionedParams(&opts, scheme.ParameterCodec).
		Do(context.Background()).
		Into(&result)

	return &result, err
}

func (c *workflowclient) Put(name string, workflow *Workflow) (*Workflow, error) {
	result := Workflow{}
	err := c.restClient.
		Put().
		Namespace(c.ns).
		Resource("workflows").
		Name(name).
		SubResource("status").
		Body(workflow).
		//VersionedParams(&opts, scheme.ParameterCodec).
		Do(context.Background()).
		Into(&result)

	return &result, err
}
