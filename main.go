package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"time"

	"github.com/kfoozminus/booklist-controller/pkg/client/clientset/versioned"
	jackinformers "github.com/kfoozminus/booklist-controller/pkg/client/informers/externalversions"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

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

	kubeclientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	jackclientset, err := versioned.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	kubeInformerFactory := informers.NewSharedInformerFactory(kubeclientset, time.Second*10)
	jackInformerFactory := jackinformers.NewSharedInformerFactory(jackclientset, time.Second*10)

	controller := NewController(kubeclientset, jackclientset, kubeInformerFactory.Apps().V1().Deployments(), jackInformerFactory.Kfoozminus().V1alpha1().Jackpots())

	stopCh := make(chan struct{})
	kubeInformerFactory.Start(stopCh)
	jackInformerFactory.Start(stopCh)

	if err = controller.Run(1, stopCh); err != nil {
		fmt.Errorf("error")
	}
}
