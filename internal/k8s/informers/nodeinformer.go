package informers

import (
	"nginx-conf-generator/internal/k8s/types"
	"nginx-conf-generator/internal/metrics"
	"nginx-conf-generator/internal/options"
	"time"

	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// RunNodeInformer spins up a shared informer factory and fetch Kubernetes node events
func RunNodeInformer(cluster *types.Cluster, clientSet kubernetes.Interface, logger *zap.Logger, nginxConf *types.NginxConf) {
	ncgo := options.GetNginxConfGeneratorOptions()
	informerFactory := informers.NewSharedInformerFactory(clientSet, time.Second*30)
	nodeInformer := informerFactory.Core().V1().Nodes()
	nodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			node := obj.(*v1.Node)
			if val, ok := node.Labels[ncgo.WorkerNodeLabel]; !ok || val != "true" {
				logger.Debug("node is not properly annotated, skipping...")
				return
			}

			if nodeReady := isNodeReady(node); nodeReady != v1.ConditionTrue {
				logger.Debug("node is not in Ready status, skipping...")
				return
			}

			logger.Info("adding node to the cluster.Workers", zap.String("node", node.Status.Addresses[0].Address))
			worker := types.NewWorker(cluster.MasterIP, node.Status.Addresses[0].Address, v1.ConditionTrue)

			// add Worker to cluster.Workers slice
			addWorker(cluster, worker)
			metrics.TargetNodeCounter.Inc()

			// add Worker to each nodePort.Workers in the cluster.NodePorts slice
			cluster.Mu.Lock()
			addWorkerToNodePorts(cluster.NodePorts, worker)
			cluster.Mu.Unlock()

			if err := applyChanges(ncgo, nginxConf); err != nil {
				logger.Fatal("fatal error occured while applying changes", zap.String("error", err.Error()))
			}
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			oldNode := oldObj.(*v1.Node)
			newNode := newObj.(*v1.Node)

			// check if it's a real update
			if oldNode.ResourceVersion == newNode.ResourceVersion {
				logger.Debug("not a real update, skipping")
				return
			}

			_, newOk := newNode.Labels[ncgo.WorkerNodeLabel]
			oldWorker := types.NewWorker(cluster.MasterIP, oldNode.Status.Addresses[0].Address, isNodeReady(oldNode))
			newWorker := types.NewWorker(cluster.MasterIP, newNode.Status.Addresses[0].Address, isNodeReady(newNode))

			if newWorker.NodeCondition != v1.ConditionTrue || !newOk {
				logger.Debug("updated node is either not Ready or not annotated")
				if i, found := findWorker(cluster.Workers, oldWorker); found {
					logger.Info("node is not healthy or is not labelled, removing from cluster.Workers!",
						zap.String("node", oldNode.Name))
					removeWorkerFromCluster(cluster, i)
					removeWorkerFromNodePorts(cluster.NodePorts, oldWorker)

					logger.Info("successfully removed node from each nodePort in the cluster.NodePorts",
						zap.String("node", oldNode.Name))
					metrics.TargetNodeCounter.Desc()
					if err := applyChanges(ncgo, nginxConf); err != nil {
						logger.Fatal("fatal error occured while applying changes", zap.String("error", err.Error()))
					}
				}
				return
			}

			if _, found := findWorker(cluster.Workers, oldWorker); !found {
				nodeReady := isNodeReady(newNode)
				logger.Info("adding node to the cluster.Workers", zap.String("node", newNode.Status.Addresses[0].Address))
				worker := types.NewWorker(cluster.MasterIP, newNode.Status.Addresses[0].Address, nodeReady)

				// add Worker to cluster.Workers slice
				addWorker(cluster, worker)
				metrics.TargetNodeCounter.Inc()

				// add Worker to each nodePort.Workers in the cluster.NodePorts slice
				cluster.Mu.Lock()
				addWorkerToNodePorts(cluster.NodePorts, worker)
				cluster.Mu.Unlock()
				if err := applyChanges(ncgo, nginxConf); err != nil {
					logger.Fatal("fatal error occured while applying changes", zap.String("error", err.Error()))
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			node := obj.(*v1.Node)
			nodeReady := isNodeReady(node)
			worker := types.NewWorker(cluster.MasterIP, node.Status.Addresses[0].Address, nodeReady)
			logger.Info("delete event fetched for node", zap.String("node", node.Name))
			index, found := findWorker(cluster.Workers, worker)
			if found {
				logger.Info("node found in the cluster.Workers, removing...", zap.String("node", node.Name))
				removeWorkerFromCluster(cluster, index)
				logger.Info("successfully removed node from cluster.Workers slice!", zap.String("node", node.Name))

				removeWorkerFromNodePorts(cluster.NodePorts, worker)
				logger.Info("successfully removed node from each nodePort in the cluster.NodePorts", zap.String("node", node.Name))
				metrics.TargetNodeCounter.Desc()

				if err := applyChanges(ncgo, nginxConf); err != nil {
					logger.Fatal("fatal error occured while applying changes", zap.String("error", err.Error()))
				}
			} else {
				logger.Debug("node not found in the cluster.workers, skipping remove operation",
					zap.String("node", node.Name))
			}
		},
	})
	informerFactory.Start(wait.NeverStop)
	informerFactory.WaitForCacheSync(wait.NeverStop)
}
