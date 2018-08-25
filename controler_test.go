package main

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubeinformers "k8s.io/client-go/informers"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

var (
	alwaysReady        = func() bool { return true }
	noResyncPeriodFunc = func() time.Duration { return 0 }
)

type fixture struct {
	t          *testing.T
	kubeclient *k8sfake.Clientset
	// Objects to put in the store.
	nodesLister []*corev1.Node
	actions     []string
	// Objects from here preloaded into NewSimpleFake.
	kubeobjects []runtime.Object
}

func newFixture(t *testing.T) *fixture {
	f := &fixture{}
	f.t = t
	f.kubeobjects = []runtime.Object{}

	return f
}

func (f *fixture) newController(ipUpdater *MockIPUpdater) (*NodeController, kubeinformers.SharedInformerFactory) {
	f.kubeclient = k8sfake.NewSimpleClientset(f.kubeobjects...)

	k8sI := kubeinformers.NewSharedInformerFactory(f.kubeclient, noResyncPeriodFunc())

	c := NewNodeController(f.kubeclient, k8sI.Core().V1().Nodes(), ipUpdater)

	c.nodesSynced = alwaysReady
	c.recorder = &record.FakeRecorder{}

	for _, d := range f.nodesLister {
		k8sI.Core().V1().Nodes().Informer().GetIndexer().Add(d)
	}

	return c, k8sI
}

func (f *fixture) run(name string) {
	f.runController(name, true, false)
}

func getKey(node *corev1.Node, t *testing.T) string {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(node)
	if err != nil {
		t.Errorf("Unexpected error getting key for node %v: %v", node.Name, err)
		return ""
	}
	return key
}

type MockIPUpdater struct {
	actions []string
}

func (m *MockIPUpdater) CreateOrUpdateVMPulicIP(ctx context.Context, vmName string, ipName string) error {
	m.actions = append(m.actions, "IP_CREATE")
	return nil
}
func (m *MockIPUpdater) DeletePublicIP(ctx context.Context, ipName string) error {
	m.actions = append(m.actions, "IP_DELETE")
	return nil
}
func (m *MockIPUpdater) DisassociatePublicIPForNode(ctx context.Context, nodeName string) error {
	m.actions = append(m.actions, "IP_DISASSOCIATE")
	return nil
}

func TestAddNode(t *testing.T) {

	f := newFixture(t)

	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "testNode"}}

	f.nodesLister = append(f.nodesLister, node)
	f.kubeobjects = append(f.kubeobjects, node)

	f.expectCreateIPAction()
	f.run(getKey(node, t))

}

func TestDeleteNode(t *testing.T) {

	f := newFixture(t)

	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "testNode"}}

	f.expectDeleteIPAction()
	f.run(getKey(node, t))

}

func (f *fixture) expectCreateIPAction() {
	f.actions = append(f.actions, "IP_CREATE")
}
func (f *fixture) expectDeleteIPAction() {
	f.actions = append(f.actions, "IP_DELETE")
}

func (f *fixture) runController(nodeName string, startInformers bool, expectError bool) {
	ipUpdater := &MockIPUpdater{actions: []string{}}
	c, k8sI := f.newController(ipUpdater)
	if startInformers {
		stopCh := make(chan struct{})
		defer close(stopCh)
		k8sI.Start(stopCh)
	}

	err := c.syncHandler(nodeName)
	if !expectError && err != nil {
		f.t.Errorf("error syncing node: %v", err)
	} else if expectError && err == nil {
		f.t.Error("expected error syncing node, got nil")
	}

	for i, action := range ipUpdater.actions {
		if len(f.actions) < i+1 {
			f.t.Errorf("%d unexpected actions: %+v", len(ipUpdater.actions)-len(f.actions), ipUpdater.actions[i:])
			break
		}
		expectedAction := f.actions[i]

		if action != expectedAction {
			f.t.Errorf("Different expected actions, action %v vs expectedAction %v", action, expectedAction)
		}
	}

	if len(f.actions) > len(ipUpdater.actions) {
		f.t.Errorf("%d additional expected actions:%+v", len(f.actions)-len(ipUpdater.actions), f.actions[len(ipUpdater.actions):])
	}

}
