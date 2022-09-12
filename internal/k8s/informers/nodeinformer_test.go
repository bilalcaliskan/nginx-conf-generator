package informers

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/bilalcaliskan/nginx-conf-generator/internal/k8s/types"
	"github.com/bilalcaliskan/nginx-conf-generator/internal/logging"
	"github.com/bilalcaliskan/nginx-conf-generator/internal/options"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
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

func (fAPI *FakeAPI) updateNode(name string, status v1.ConditionStatus, isLabelled bool, version string) (*v1.Node, error) {
	node, _ := fAPI.getNode(name)
	node.Status.Conditions[0].Status = status
	node.ResourceVersion = version

	if !isLabelled {
		delete(node.Labels, opts.WorkerNodeLabel)
	} else {
		node.Labels[opts.WorkerNodeLabel] = ""
	}

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

func (fAPI *FakeAPI) createNode(name, ip string, isReady v1.ConditionStatus, isLabelled bool) (*v1.Node, error) {
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
					Status: isReady,
					Reason: "KubeletReady",
				},
			},
			Addresses: []v1.NodeAddress{
				{Type: v1.NodeInternalIP, Address: ip},
				{Type: v1.NodeHostName, Address: name},
			},
		},
	}

	if isLabelled {
		node.ObjectMeta.Labels[opts.WorkerNodeLabel] = "true"
	}

	ctx, cancel := context.WithTimeout(parentCtx, 60*time.Second)
	defer cancel()
	node, err := fAPI.ClientSet.CoreV1().Nodes().Create(ctx, node, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	return node, nil
}

func TestRunNodeInformer(t *testing.T) {
	api := getFakeAPI()
	assert.NotNil(t, api)

	opts.Mu.Lock()
	opts.TemplateInputFile = "../../../resources/ncg.conf.tmpl"
	opts.TemplateOutputFile = "/etc/nginx/conf.d/ncg_test.conf"
	opts.Mu.Unlock()

	var clusters []*types.Cluster
	nginxConf := types.NewNginxConf(clusters)
	cluster := types.NewCluster("", make([]*types.Worker, 0))
	nginxConf.Clusters = append(nginxConf.Clusters, cluster)

	go func() {
		RunNodeInformer(cluster, api.ClientSet, logging.GetLogger(), nginxConf)
	}()

	cases := []struct {
		caseName                               string
		isLabelledOnCreate, isLabelledOnUpdate bool
		isReadyOnCreate, isReadyOnUpdate       v1.ConditionStatus
	}{
		{"case1", true, true, v1.ConditionTrue, v1.ConditionTrue},
		{"case2", true, true, v1.ConditionFalse, v1.ConditionTrue},
		{"case3", true, false, v1.ConditionTrue, v1.ConditionTrue},
		{"case4", true, true, v1.ConditionTrue, v1.ConditionFalse},
	}

	for _, tc := range cases {
		t.Run(tc.caseName, func(t *testing.T) {
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				time.Sleep(2 * time.Second)
				defer wg.Done()
				node1, err1 := api.createNode("node01", "10.0.0.44", tc.isReadyOnCreate, tc.isLabelledOnCreate)
				assert.Nil(t, err1)
				assert.NotNil(t, node1)

				_, _ = api.createNode("node02", "10.0.0.45", v1.ConditionTrue, true)
			}()
			wg.Wait()

			time.Sleep(2 * time.Second)

			for i := 0; i < 5; i++ {
				port, _ := strconv.ParseInt(fmt.Sprintf("3044%d", i), 10, 32)
				cluster.Mu.Lock()
				nodeport := &types.NodePort{
					MasterIP: "",
					Port:     int32(port),
					Workers:  cluster.Workers,
				}
				addNodePort(&cluster.NodePorts, nodeport)
				cluster.Mu.Unlock()
			}

			wg.Add(1)
			go func() {
				time.Sleep(2 * time.Second)
				defer wg.Done()
				updatedNode, err := api.updateNode("node01", tc.isReadyOnUpdate, tc.isLabelledOnUpdate, "123456")
				assert.NotNil(t, updatedNode)
				assert.Nil(t, err)
				t.Logf("node updated")
			}()
			wg.Wait()

			wg.Add(1)
			go func() {
				time.Sleep(2 * time.Second)
				defer wg.Done()
				node, err := api.getNode("node01")
				assert.Nil(t, err)
				assert.NotNil(t, node)
				t.Logf("node fetched")
				assert.Equal(t, tc.isReadyOnUpdate, node.Status.Conditions[0].Status)
			}()
			wg.Wait()

			node, err := api.getNode("node01")
			assert.Nil(t, err)
			assert.NotNil(t, node)

			wg.Add(1)
			go func() {
				time.Sleep(2 * time.Second)
				defer wg.Done()
				err := api.deleteNode("node01")
				assert.Nil(t, err)
				t.Logf("node deleted")
			}()
			wg.Wait()

			wg.Add(1)
			go func() {
				time.Sleep(2 * time.Second)
				defer wg.Done()
				worker := types.NewWorker("", node.Name, v1.ConditionTrue)
				for {
					cluster.Mu.Lock()
					_, found := findWorker(cluster.Workers, worker)
					cluster.Mu.Unlock()
					if !found {
						t.Logf("node delete test succeeded")
						break
					}
				}
			}()
			wg.Wait()
		})
	}
}
