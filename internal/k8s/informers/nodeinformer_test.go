package informers

import (
	"context"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"nginx-conf-generator/internal/k8s/types"
	"nginx-conf-generator/internal/logging"
	"nginx-conf-generator/internal/options"
	"sync"
	"testing"
	"time"
)

var (
	parentCtx = context.Background()
	opts      = options.GetNginxConfGeneratorOptions()
)

type FakeAPI struct {
	ClientSet kubernetes.Interface
	Namespace string
}

func getFakeAPI() *FakeAPI {
	client := fake.NewSimpleClientset()
	api := &FakeAPI{ClientSet: client, Namespace: "default"}
	return api
}

func (fAPI *FakeAPI) deleteNode(name string) error {
	// node, _ := fAPI.getNode(name)

	// gracePeriodSeconds := int64(0)
	ctx, cancel := context.WithTimeout(parentCtx, 60*time.Second)
	defer cancel()
	return fAPI.ClientSet.CoreV1().Nodes().Delete(ctx, name, metav1.DeleteOptions{})
}

func (fAPI *FakeAPI) updateNode(name string) (*v1.Node, error) {
	node, _ := fAPI.getNode(name)
	node.Status.Conditions[0].Status = "False"
	node.ResourceVersion = "123456"

	ctx, cancel := context.WithTimeout(parentCtx, 60*time.Second)
	defer cancel()
	node, err := fAPI.ClientSet.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}

	return node, nil
}

func (fAPI *FakeAPI) getNode(name string) (*v1.Node, error) {
	ctx, cancel := context.WithTimeout(parentCtx, 60*time.Second)
	defer cancel()
	return fAPI.ClientSet.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
}

func (fAPI *FakeAPI) createNode(name string) (*v1.Node, error) {
	node := &v1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"beta.kubernetes.io/arch": "amd64",
				"beta.kubernetes.io/os":   "linux",
				"kubernetes.io/arch":      "amd64",
				"kubernetes.io/hostname":  name,
				"kubernetes.io/os":        "linux",
				"worker":                  "",
			},
		},
		Spec: v1.NodeSpec{
			PodCIDR: "10.244.1.0/24",
			PodCIDRs: []string{
				"10.244.1.0/24",
			},
		},
		Status: v1.NodeStatus{
			Allocatable: v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("4000m"),
				v1.ResourceMemory: resource.MustParse("16378944Ki"),
			},
			NodeInfo: v1.NodeSystemInfo{
				MachineID:               "760e67beb8554645829f2357c8eb4ae7",
				SystemUUID:              "e089014d-cb57-4cd1-827f-842dbba119b5",
				BootID:                  "18a9caa0-7f70-4cf2-ad0a-d70764927dba",
				KernelVersion:           "5.15.0-kali3-amd64",
				OSImage:                 "Ubuntu 20.04.2 LTS",
				ContainerRuntimeVersion: "docker://20.10.7",
				KubeletVersion:          "v1.21.2",
				KubeProxyVersion:        "v1.21.2",
				OperatingSystem:         "linux",
				Architecture:            "amd64",
			},
			Conditions: []v1.NodeCondition{
				{
					Type:   "Ready",
					Status: "True",
					Reason: "KubeletReady",
				},
			},
			Addresses: []v1.NodeAddress{
				{Type: v1.NodeHostName, Address: name},
				{Type: v1.NodeInternalIP, Address: "192.168.49.3"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(parentCtx, 60*time.Second)
	defer cancel()
	node, err := fAPI.ClientSet.CoreV1().Nodes().Create(ctx, node, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	return node, nil
}

func TestRunNodeInformerCase1(t *testing.T) {
	/*
		- new node added with required label and required node status
		- this particular node deleted
	*/
	api := getFakeAPI()
	assert.NotNil(t, api)

	opts.Mu.Lock()
	opts.TemplateInputFile = "../../../resources/default.conf.tmpl"
	opts.Mu.Unlock()

	var clusters []*types.Cluster
	nginxConf := types.NewNginxConf(clusters)
	cluster := types.NewCluster("", make([]*types.Worker, 0))
	nginxConf.Clusters = append(nginxConf.Clusters, cluster)

	go func() {
		RunNodeInformer(cluster, api.ClientSet, logging.NewLogger(), opts, nginxConf)
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		pod, err := api.createNode("node01")
		assert.Nil(t, err)
		assert.NotNil(t, pod)
	}()
	wg.Wait()

	time.Sleep(2 * time.Second)

	node, err := api.getNode("node01")
	assert.Nil(t, err)
	assert.NotNil(t, node)

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := api.deleteNode("node01")
		assert.Nil(t, err)
	}()
	wg.Wait()

	time.Sleep(2 * time.Second)

	worker := types.NewWorker("", node.Name, "True")
	for {
		cluster.Mu.Lock()
		_, found := findWorker(cluster.Workers, *worker)
		cluster.Mu.Unlock()
		if !found {
			break
		}
	}
}

func TestRunNodeInformerCase2(t *testing.T) {
	/*
		- new node added with required label and required node status
		- this particular node's node status updated as NotReady
	*/
	createChan := make(chan bool, 1)
	updateChan := make(chan bool, 1)
	deleteChan := make(chan bool, 1)

	api := getFakeAPI()
	assert.NotNil(t, api)

	opts.Mu.Lock()
	opts.TemplateInputFile = "../../../resources/default.conf.tmpl"
	opts.Mu.Unlock()

	var clusters []*types.Cluster
	nginxConf := types.NewNginxConf(clusters)
	cluster := types.NewCluster("", make([]*types.Worker, 0))
	nginxConf.Clusters = append(nginxConf.Clusters, cluster)

	go func() {
		RunNodeInformer(cluster, api.ClientSet, logging.NewLogger(), opts, nginxConf)
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		pod, err := api.createNode("node01")
		assert.Nil(t, err)
		assert.NotNil(t, pod)
		createChan <- true
	}()
	wg.Wait()

	/*	wg.Add(1)
		go func() {
			defer wg.Done()
			node, err := api.getNode("node01")
			assert.Nil(t, err)
			assert.NotNil(t, node)
			for {
				if node.Status.Conditions[0].Status == "True" {
					break
				}
				time.Sleep(1 * time.Second)
			}
		}()
		wg.Wait()*/

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-createChan
		updatedNode, err := api.updateNode("node01")
		assert.NotNil(t, updatedNode)
		assert.Nil(t, err)
		updateChan <- true
	}()
	wg.Wait()

	//time.Sleep(2 * time.Second)
	/*	wg.Add(1)
		go func() {
			defer wg.Done()
			node, err := api.getNode("node01")
			assert.Nil(t, err)
			assert.NotNil(t, node)
			for {
				if node.Status.Conditions[0].Status == "False" {
					break
				}
				time.Sleep(1 * time.Second)
			}
		}()
		wg.Wait()*/

	node, err := api.getNode("node01")
	assert.Nil(t, err)
	assert.NotNil(t, node)

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-updateChan
		err := api.deleteNode("node01")
		assert.Nil(t, err)
		deleteChan <- true
	}()
	wg.Wait()

	//time.Sleep(2 * time.Second)

	worker := types.NewWorker("", node.Name, "True")
	for {
		cluster.Mu.Lock()
		_, found := findWorker(cluster.Workers, *worker)
		cluster.Mu.Unlock()
		if !found {
			break
		}
	}
}
