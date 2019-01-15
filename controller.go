package main

import (
	"fmt"
	"time"

	kfoozminusv1alpha1 "github.com/kfoozminus/booklist-controller/pkg/apis/kfoozminus/v1alpha1"
	"github.com/kfoozminus/booklist-controller/pkg/client/clientset/versioned"
	informerv1alpha1 "github.com/kfoozminus/booklist-controller/pkg/client/informers/externalversions/kfoozminus/v1alpha1"
	listerv1alpha1 "github.com/kfoozminus/booklist-controller/pkg/client/listers/kfoozminus/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	informerv1 "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	listerv1 "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

type Controller struct {
	kubeInterface kubernetes.Interface
	jackInterface versioned.Interface

	deploymentsLister listerv1.DeploymentLister
	deploymentsSynced cache.InformerSynced

	jackpotsLister listerv1alpha1.JackpotLister
	jackpotsSynced cache.InformerSynced

	workqueue workqueue.RateLimitingInterface
}

func NewController(
	kubeclientset kubernetes.Interface, jackclientset versioned.Interface,
	deploymentInformer informerv1.DeploymentInformer, jackpotInformer informerv1alpha1.JackpotInformer) *Controller {

	controller := &Controller{
		kubeInterface:     kubeclientset,
		jackInterface:     jackclientset,
		deploymentsLister: deploymentInformer.Lister(),
		deploymentsSynced: deploymentInformer.Informer().HasSynced,
		jackpotsLister:    jackpotInformer.Lister(),
		jackpotsSynced:    jackpotInformer.Informer().HasSynced,
		workqueue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "JackJack"),
	}

	jackpotInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleJackpot,
		UpdateFunc: func(old, new interface{}) {
			controller.handleJackpot(new)
		},
	})
	deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleDeployment,
		UpdateFunc: func(old, new interface{}) {
			newDeployment := new.(*appsv1.Deployment)
			oldDeployment := old.(*appsv1.Deployment)
			if newDeployment.ResourceVersion == oldDeployment.ResourceVersion {
				return
			}
			controller.handleDeployment(new)
		},
		DeleteFunc: controller.handleDeployment,
	})

	return controller
}

func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	fmt.Println("Starting Controller...")

	if ok := cache.WaitForCacheSync(stopCh, c.deploymentsSynced, c.jackpotsSynced); !ok {
		return fmt.Errorf("Timed out waiting for caches to sync")
	}

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	return nil

}

func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()
	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var key string
		var ok bool

		if key, ok = obj.(string); !ok {
			c.workqueue.Forget(obj)
			fmt.Errorf("Invalid key")
			return nil
		}

		if err := c.syncHandler(key); err != nil {
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error while syncing %s", key)
		}

		//c.workqueue.Forget(key)
		c.workqueue.Forget(obj)
		fmt.Println("Successfully synced", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}
	return true
}

func (c *Controller) syncHandler(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Invalid resource key %s", key))
		return nil
	}

	jackpot, err := c.jackpotsLister.Jackpots(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("foo %s in workqueue no longer exists", key))
			return nil
		}
		return err
	}

	deploymentName := jackpot.Spec.DeploymentName
	fmt.Println(deploymentName)
	if deploymentName == "" {
		utilruntime.HandleError(fmt.Errorf("%s: deployment name must be specified", key))
		return nil
	}

	deployment, err := c.deploymentsLister.Deployments(jackpot.Namespace).Get(deploymentName)
	if errors.IsNotFound(err) {
		deployment, err = c.kubeInterface.AppsV1().Deployments(jackpot.Namespace).Create(newDeployment(jackpot))
	}

	if err != nil {
		return err
	}

	if !metav1.IsControlledBy(deployment, jackpot) {
		return fmt.Errorf("Resource already exists and is not managed by Jackpot")
	}

	if jackpot.Spec.Replicas != nil && *jackpot.Spec.Replicas != *deployment.Spec.Replicas {
		deployment, err = c.kubeInterface.AppsV1().Deployments(jackpot.Namespace).Update(newDeployment(jackpot))
	}

	if err != nil {
		return err
	}

	err = c.updateStatus(jackpot, deployment)
	if err != nil {
		return err
	}

	return nil
}

func (c *Controller) updateStatus(jackpot *kfoozminusv1alpha1.Jackpot, deployment *appsv1.Deployment) error {
	jackpotCopy := jackpot.DeepCopy()
	jackpotCopy.Status.AvailableReplicas = deployment.Status.AvailableReplicas
	_, err := c.jackInterface.KfoozminusV1alpha1().Jackpots(jackpot.Namespace).Update(jackpotCopy)
	return err
}

func (c *Controller) handleJackpot(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}

func (c *Controller) handleDeployment(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		fmt.Printf("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	fmt.Printf("Processing object: %s\n", object.GetName())
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		if ownerRef.Kind != "Jackpot" {
			return
		}

		jackpot, err := c.jackpotsLister.Jackpots(object.GetNamespace()).Get(ownerRef.Name)
		if err != nil {
			klog.V(4).Infof("ignoring orphaned object '%s' of foo '%s'", object.GetSelfLink(), ownerRef.Name)
			return
		}

		c.handleJackpot(jackpot)
		return
	}
}

func newDeployment(jackpot *kfoozminusv1alpha1.Jackpot) *appsv1.Deployment {
	labels := map[string]string{
		"app":        "booklist-controller",
		"controller": jackpot.Name,
	}
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jackpot.Spec.DeploymentName,
			Namespace: jackpot.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(jackpot, schema.GroupVersionKind{
					Group:   kfoozminusv1alpha1.SchemeGroupVersion.Group,
					Version: kfoozminusv1alpha1.SchemeGroupVersion.Version,
					Kind:    "Jackpot",
				}),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: jackpot.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "booklist-controller",
							Image: jackpot.Spec.Image,
						},
					},
				},
			},
		},
	}
}
