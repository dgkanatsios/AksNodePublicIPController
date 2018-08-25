package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	informercorev1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	listercorev1 "k8s.io/client-go/listers/core/v1"

	helpers "github.com/dgkanatsios/AksNodePublicIPController/helpers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

const controllerAgentName = "nodes-controller"

const (
	successSynced         = "Synced"
	successCreatingIP     = "SuccessCreatingIP"
	successDeletingIP     = "SuccessDeletingIP"
	errResourceExists     = "ErrResourceExists"
	messageResourceSynced = "Node synced successfully"
	errorCreatingIP       = "ErrorCreatingIP"
	errorDeletingIP       = "ErrorDeletingIP"
)

var ctx = context.Background()

// NodeController is the Node Controller
type NodeController struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface

	nodesLister listercorev1.NodeLister
	nodesSynced cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder

	ipUpdater helpers.IPUpdater
}

// NewNodeController returns a new sample controller
func NewNodeController(
	kubeclientset kubernetes.Interface,
	nodeInformer informercorev1.NodeInformer, ipARMUpdater helpers.IPUpdater) *NodeController {

	// Create event broadcaster
	// Add sample-controller types to the default Kubernetes Scheme so Events can be
	// logged for sample-controller types.
	log.Info("Creating event broadcaster for Node-Public IP controller")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(log.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &NodeController{
		kubeclientset: kubeclientset,
		nodesLister:   nodeInformer.Lister(),
		nodesSynced:   nodeInformer.Informer().HasSynced,
		workqueue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Nodes"),
		recorder:      recorder,
		ipUpdater:     ipARMUpdater,
	}

	log.Info("Setting up event handlers for Node-Public IP controller")
	// Set up an event handler for when Node resources change

	// Set up an event handler for when Node resources change. This
	// handler will lookup the owner of the given Node, and it will enqueue that Node resource for
	// processing. This way, we don't need to implement custom logic for
	// handling Node resources. More info on this pattern:
	// https://github.com/kubernetes/community/blob/8cafef897a22026d42f5e5bb3f104febe7e29830/contributors/devel/controllers.md
	nodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newNode := new.(*corev1.Node)
			oldNode := old.(*corev1.Node)
			if newNode.ResourceVersion == oldNode.ResourceVersion {
				// Periodic resync will send update events for all known Nodes.
				// Two different versions of the same Node will always have different RVs.
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})

	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *NodeController) Run(threadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	log.Info("Starting Node-Public IP controller")

	// Wait for the caches to be synced before starting workers
	log.Info("Waiting for informer caches to sync for Node-Public IP controller")
	if ok := cache.WaitForCacheSync(stopCh, c.nodesSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync for Node-Public IP controller")
	}

	log.Info("Starting workers for Node-Public IP controller")
	// Launch workers to process Node resources
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	log.Info("Started workers for Node-Public IP controller")
	<-stopCh
	log.Info("Shutting down workers for Node-Public IP controller")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *NodeController) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *NodeController) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// Node resource to be synced.
		if err := c.syncHandler(key); err != nil {
			return fmt.Errorf("error syncing '%s': %s", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		//log.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Node resource
// with the current status of the resource.
func (c *NodeController) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	_, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	//Get the Node with this name

	node, err := c.nodesLister.Get(name)

	if err != nil {
		// The Node resource may no longer exist, in which case we delete the Public IP and stop
		// processing.
		if errors.IsNotFound(err) {
			runtime.HandleError(fmt.Errorf("Node '%s' in work queue no longer exists in Node-Public IP controller", name))
			errDelete := c.deletePublicIPForNode(name)
			if errDelete != nil {
				log.Infof("Error deleting IP for Node %s: %v", name, errDelete.Error())
				return errDelete
			}
			log.Infof("Successfully deleted IP for Node %s", name)
			return nil
		}

		return err // cannot list nodes
	}

	if !nodeHasPublicIP(node) {
		//node does not have a Public IP
		log.Infof("Node with name %s does not have a Public IP, trying to create one", node.Name)
		err := c.ipUpdater.CreateOrUpdateVMPulicIP(ctx, node.Name, helpers.GetPublicIPName(node.Name))
		if err != nil {
			runtime.HandleError(fmt.Errorf("Error in creating IP %s, for Node %s", err.Error(), node.Name))
			c.recorder.Event(node, corev1.EventTypeWarning, errorCreatingIP, err.Error())
			return nil
		}
		c.recorder.Event(node, corev1.EventTypeNormal, successCreatingIP, fmt.Sprintf("Successfully created IP for Node %s", node.Name))
	}

	//c.recorder.Event(node, corev1.EventTypeNormal, successSynced, messageResourceSynced)
	return nil
}

func (c *NodeController) handleObject(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			runtime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			runtime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		log.Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	//log.Infof("Processing object: %s", object.GetName())

	c.enqueueNode(object)
}

func (c *NodeController) deletePublicIPForNode(nodeName string) error {
	log.Infof("Node with name %s has been deleted, trying to delete its Public IP", nodeName)
	err := c.ipUpdater.DeletePublicIP(ctx, helpers.GetPublicIPName(nodeName))

	// there is a chance that NIC is still alive so IP Address is still associated and we'll get an error
	// this is the error message:
	// Failure sending request: StatusCode=0 -- Original Error: autorest/azure: Service returned an error. Status=400 Code="PublicIPAddressCannotBeDeleted" Message="Public IP address /subscriptions/XXX/resourceGroups/XXX/providers/Microsoft.Network/publicIPAddresses/XXX can not be deleted since it is still allocated
	if err != nil && strings.Contains(err.Error(), `Code="PublicIPAddressCannotBeDeleted"`) {
		// try to disassociate the Public IP
		errDis := c.ipUpdater.DisassociatePublicIPForNode(ctx, nodeName)
		if errDis != nil {
			runtime.HandleError(fmt.Errorf("Cannot disassociate Public IP for node %s due to error %s", nodeName, errDis.Error()))
		}
		// regardless of whether we get an error in disassociating, we should try and delete the Public IP again
		errDeleteIP := c.ipUpdater.DeletePublicIP(ctx, helpers.GetPublicIPName(nodeName))
		if errDeleteIP != nil {
			runtime.HandleError(fmt.Errorf("Could not delete Public IP for node %s due to error %s", nodeName, errDeleteIP.Error()))
			return errDeleteIP
		}

	} else if err != nil {
		runtime.HandleError(fmt.Errorf("Could not delete Public IP for node %s due to error %s", nodeName, err.Error()))
		return err
	}

	log.Infof("Successfully deleted Public IP for Node with name %s", nodeName)
	return nil
}

// enqueuePod takes a Pod resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Pod.
func (c *NodeController) enqueueNode(obj interface{}) {

	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}

// returns true if the Node has a Public IP
func nodeHasPublicIP(node *corev1.Node) bool {
	for _, x := range node.Status.Addresses {
		if x.Type == corev1.NodeExternalIP {
			//write down node's Public IP
			//log.Printf("Node %s has a Public IP: %s", node.Name, x.Address)
			return true
		}
	}
	return false
}
