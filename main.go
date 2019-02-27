package main

import (
	"flag"
	"os"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"

	log "github.com/Sirupsen/logrus"

	"github.com/dgkanatsios/AksNodePublicIPController/pkg/helpers"
	"github.com/dgkanatsios/AksNodePublicIPController/pkg/signals"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	informers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
)

var (
	masterURL  string
	kubeconfig string
)

const (
	configMapName = "leaderlockpublicip"
)

func main() {
	id, err := os.Hostname()
	if err != nil {
		log.Fatalf("cannot get hostname because of %s", err.Error())
	}

	err = helpers.InitializeServicePrincipalDetails()

	if err != nil {
		log.Fatalf("cannot initialize Service Principal credentials: %s", err.Error())
	}

	flag.Parse()

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	var config *rest.Config
	if len(kubeconfig) > 0 {
		config, err = clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	} else {
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	namespace := os.Getenv("POD_NAMESPACE")
	if namespace == "" {
		namespace = metav1.NamespaceDefault
	}

	// use a client that will stop allowing new requests once the context ends
	//config.Wrap(transport.ContextCanceller(ctx, fmt.Errorf("the leader is shutting down")))
	kubeClient := kubernetes.NewForConfigOrDie(config)

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(log.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: id})

	// we use the ConfigMap lock type since edits to ConfigMaps are less common
	// and fewer objects in the cluster watch "all ConfigMaps" (unlike the older
	// Endpoints lock type, where quite a few system agents like the kube-proxy
	// and ingress controllers must watch endpoints).
	lock := &resourcelock.ConfigMapLock{
		ConfigMapMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      configMapName,
		},
		Client: kubeClient.CoreV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity:      id,
			EventRecorder: recorder,
		},
	}

	// start the leader election code loop
	leaderelection.RunOrDie(leaderelection.LeaderElectionConfig{
		Lock:          lock,
		LeaseDuration: 60 * time.Second,
		RenewDeadline: 15 * time.Second,
		RetryPeriod:   5 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(_ <-chan struct{}) {
				// we're notified when we start - this is where you would
				// usually put your code
				log.Printf("%s: leading", id)
				sharedInformers := informers.NewSharedInformerFactory(kubeClient, 10*time.Minute)

				controller := NewNodeController(kubeClient, sharedInformers.Core().V1().Nodes(), &helpers.IPUpdate{})

				go sharedInformers.Start(stopCh)

				if err = controller.Run(1, stopCh); err != nil {
					log.Fatalf("Error running controller: %s", err.Error())
				}
			},
			OnStoppedLeading: func() {
				// we can do cleanup here, or after the RunOrDie method
				// returns
				log.Printf("%s: lost", id)
			},
		},
	})

	// because the context is closed, the client should report errors
	_, err = kubeClient.CoreV1().ConfigMaps(namespace).Get(configMapName, metav1.GetOptions{})
	if err == nil || !strings.Contains(err.Error(), "the leader is shutting down") {
		log.Fatalf("%s: expected to get an error when trying to make a client call: %v", id, err)
	}

	// we no longer hold the lease, so perform any cleanup and then
	// exit
	log.Printf("%s: done", id)
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
}
