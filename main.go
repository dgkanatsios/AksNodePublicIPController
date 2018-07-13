package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/dgkanatsios/AksNodePublicIPController/pkg/signals"
	informers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	masterURL  string
	kubeconfig string
)

func readServicePrincipalDetails() {
	file, e := ioutil.ReadFile("/aks/azure.json")
	if e != nil {
		fmt.Printf("File error: %v\n", e)
		os.Exit(1)
	}
	var f interface{}
	err := json.Unmarshal(file, &f)

	if err != nil {
		fmt.Printf("Unmarshaling error: %v\n", err)
		os.Exit(1)
	}

	m := f.(map[string]interface{})

	fmt.Println("%s", m["tenantId"])
	fmt.Println("%s", m["subscriptionId"])
	fmt.Println("%s", m["aadClientId"])
	fmt.Println("%s", m["aadClientSecret"])
	fmt.Println("%s", m["resourceGroup"])

}

func main() {

	readServicePrincipalDetails()

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

	sharedInformers := informers.NewSharedInformerFactory(kubeClient, 30*time.Minute)

	controller := NewController(kubeClient, sharedInformers.Core().V1().Nodes())

	go sharedInformers.Start(stopCh)

	if err = controller.Run(2, stopCh); err != nil {
		log.Fatalf("Error running controller: %s", err.Error())
	}
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
}

/*
/etc/kubernetes/azure.json is ...

{
    "cloud":"AzurePublicCloud",
    "tenantId": "XXX",
    "subscriptionId": "XXX",
    "aadClientId": "XXXX",
    "aadClientSecret": "XXXXX",
    "resourceGroup": "MC_akslala_akslala_westeurope",
    "location": "westeurope",
	...
}
*/
