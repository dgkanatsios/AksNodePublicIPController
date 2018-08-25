package main

import (
	"flag"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/dgkanatsios/AksNodePublicIPController/helpers"
	"github.com/dgkanatsios/AksNodePublicIPController/pkg/signals"
	informers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	masterURL  string
	kubeconfig string
)

func main() {

	err := helpers.InitializeServicePrincipalDetails()

	if err != nil {
		log.Fatalf("Error initializing Service Principal credentials: %s", err.Error())
	}

	flag.Parse()

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	sharedInformers := informers.NewSharedInformerFactory(kubeClient, 10*time.Minute)

	controller := NewNodeController(kubeClient, sharedInformers.Core().V1().Nodes(), &helpers.IPUpdate{})

	go sharedInformers.Start(stopCh)

	//start two workers
	if err = controller.Run(2, stopCh); err != nil {
		log.Fatalf("Error running controller: %s", err.Error())
	}
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
}
