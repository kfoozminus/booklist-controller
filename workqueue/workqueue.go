package workqueue

import (
	"flag"
	"fmt"
	"path/filepath"
	"time"

	"github.com/kfoozminus/booklist-controller/pkg/apis/kfoozminus/v1alpha1"
	"github.com/kfoozminus/booklist-controller/pkg/client/clientset/versioned"
	"k8s.io/apimachinery/pkg/fields"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/client-go/util/workqueue"
)

type Controller struct {
	indexer  cache.Indexer
	queue    workqueue.RateLimitingInterface
	informer cache.Controller
}

func NewController(queue workqueue.RateLimitingInterface, indexer cache.Indexer, informer cache.Controller) *Controller {
	return &Controller{
		informer: informer,
		indexer:  indexer,
		queue:    queue,
	}
}

func (c *Controller) Run(threadiness int, stopCh chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	go c.informer.Run(stopCh)

	if !cache.WaitForCacheSync(stopCh, c.informer.HasSynced) {
		utilruntime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runJackpotWorker, time.Second, stopCh)
	}

	<-stopCh
}

func (c *Controller) runJackpotWorker() {
	for c.processNewJackpot() {
	}
}

func (c *Controller) processNewJackpot() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}

	defer c.queue.Done(key)

	err := c.JackpotLogic(key.(string))
	c.handleErr(err, key)

	return true
}

func (c *Controller) JackpotLogic(key string) error {
	obj, exists, err := c.indexer.GetByKey(key)
	if err != nil {
		return err
	}

	if !exists {
		fmt.Println("Jackpot object has been deleted")
	} else {
		fmt.Println("Cache loading ...", obj.(*v1alpha1.Jackpot).GetName())
	}

	return nil
}

func (c *Controller) handleErr(err error, key interface{}) {
	if err == nil {
		c.queue.Forget(key)
		return
	}

	if c.queue.NumRequeues(key) < 5 {
		c.queue.AddRateLimited(key)
		return
	}

	c.queue.Forget(key)
	utilruntime.HandleError(err)
	fmt.Println("Retried 5 times - dropping the jackpot")
}

func main() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}
	clientset, err := versioned.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	jackpotWatcher := cache.NewListWatchFromClient(clientset.KfoozminusV1alpha1().RESTClient(), "jackpots", v1alpha1.NamespaceDefault, fields.Everything())
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	indexer, informer := cache.NewIndexerInformer(jackpotWatcher, &v1alpha1.Jackpot{}, 0, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
	}, cache.Indexers{})

	controller := NewController(queue, indexer, informer)

	stop := make(chan struct{})
	defer close(stop)
	go controller.Run(1, stop)
	select {}
}
