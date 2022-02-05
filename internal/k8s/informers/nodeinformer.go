package informers

import (
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"nginx-conf-generator/internal/k8s/types"
	"nginx-conf-generator/internal/metrics"
	"nginx-conf-generator/internal/options"
	"time"
)

// RunNodeInformer spins up a shared informer factory and fetch Kubernetes node events
func RunNodeInformer(cluster *types.Cluster, clientSet kubernetes.Interface, logger *zap.Logger, ncgo *options.NginxConfGeneratorOptions,
	nginxConf *types.NginxConf) {

	informerFactory := informers.NewSharedInformerFactory(clientSet, time.Second*30)
	nodeInformer := informerFactory.Core().V1().Nodes()
	nodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			node := obj.(*v1.Node)
			_, ok := node.Labels[ncgo.WorkerNodeLabel]
			nodeReady := isNodeReady(node)

			if ok && nodeReady == "True" {
				logger.Info("adding node to the cluster.Workers", zap.String("node", node.Status.Addresses[0].Address))
				worker := types.NewWorker(cluster.MasterIP, node.Status.Addresses[0].Address, nodeReady)

				// add Worker to cluster.Workers slice
				addWorker(&cluster.Workers, worker)
				metrics.TargetNodeCounter.Inc()

				// Apply changes to the template
				ncgo.Mu.Lock()
				err := renderTemplate(ncgo.TemplateInputFile, ncgo.TemplateOutputFile, nginxConf)
				ncgo.Mu.Unlock()
				if err != nil {
					logger.Fatal(ErrRenderTemplate, zap.Error(err))
				}

				// reload Nginx service
				if err = reloadNginx(); err != nil {
					logger.Fatal(ErrReloadNginx, zap.String("error", err.Error()))
				}

				logger.Info(SuccessNginxReload)
			}
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			oldNode := oldObj.(*v1.Node)
			newNode := newObj.(*v1.Node)

			// there is an update for sure
			if oldNode.ResourceVersion != newNode.ResourceVersion {
				_, newOk := newNode.Labels[ncgo.WorkerNodeLabel]
				oldWorker := types.NewWorker(cluster.MasterIP, oldNode.Status.Addresses[0].Address, isNodeReady(oldNode))
				newWorker := types.NewWorker(cluster.MasterIP, newNode.Status.Addresses[0].Address, isNodeReady(newNode))

				oldWorkerIndex, oldWorkerFound := findWorker(cluster.Workers, *oldWorker)
				if oldWorkerFound {
					if newWorker.NodeCondition == "True" && newOk {
						logger.Info("node is still healthy and labelled, skipping...",
							zap.String("node", oldWorker.HostIP))
					} else {
						logger.Info("node is not healthy or is not labelled, removing from cluster.Workers!",
							zap.String("node", oldWorker.HostIP))
						removeWorker(cluster, oldWorkerIndex)

						logger.Info("successfully removed node from each nodePort in the cluster.NodePorts",
							zap.String("node", oldWorker.HostIP))
						metrics.TargetNodeCounter.Desc()

						// Apply changes to the template
						ncgo.Mu.Lock()
						err := renderTemplate(ncgo.TemplateInputFile, ncgo.TemplateOutputFile, nginxConf)
						ncgo.Mu.Unlock()
						if err != nil {
							logger.Fatal(ErrRenderTemplate, zap.Error(err))
						}

						// reload Nginx service
						if err = reloadNginx(); err != nil {
							logger.Fatal(ErrReloadNginx, zap.String("error", err.Error()))
						}

						logger.Info(SuccessNginxReload)
					}
				} else {
					if newWorker.NodeCondition == "True" && newOk {
						logger.Info("node is now healthy and labeled, adding to the cluster.Workers slice",
							zap.String("node", oldWorker.HostIP))
						addWorker(&cluster.Workers, newWorker)

						// add NewWorker to each nodePort.Workers in the cluster.NodePorts slice
						// addWorkerToNodePorts(cluster.NodePorts, newWorker)
						metrics.TargetNodeCounter.Inc()

						// Apply changes to the template
						ncgo.Mu.Lock()
						err := renderTemplate(ncgo.TemplateInputFile, ncgo.TemplateOutputFile, nginxConf)
						ncgo.Mu.Unlock()
						if err != nil {
							logger.Fatal(ErrRenderTemplate, zap.Error(err))
						}

						// reload Nginx service
						if err = reloadNginx(); err != nil {
							logger.Fatal(ErrReloadNginx, zap.String("error", err.Error()))
						}

						logger.Info(SuccessNginxReload)
					} else {
						logger.Info("node is still unhealthy or unlabelled, skipping...",
							zap.String("node", oldWorker.HostIP))
					}
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			node := obj.(*v1.Node)
			nodeReady := isNodeReady(node)
			worker := types.NewWorker(cluster.MasterIP, node.Status.Addresses[0].Address, nodeReady)
			logger.Info("delete event fetched for node", zap.String("node", worker.HostIP))
			index, found := findWorker(cluster.Workers, *worker)
			if found {
				logger.Info("node found in the cluster.Workers, removing...", zap.String("node", worker.HostIP))
				removeWorker(cluster, index)
				logger.Info("successfully removed node from cluster.Workers slice!", zap.String("node", worker.HostIP))

				logger.Info("successfully removed node from each nodePort in the cluster.NodePorts", zap.String("node", worker.HostIP))
				metrics.TargetNodeCounter.Desc()

				// Apply changes to the template
				ncgo.Mu.Lock()
				err := renderTemplate(ncgo.TemplateInputFile, ncgo.TemplateOutputFile, nginxConf)
				ncgo.Mu.Unlock()
				if err != nil {
					logger.Fatal(ErrRenderTemplate, zap.Error(err))
				}

				// reload Nginx service
				if err = reloadNginx(); err != nil {
					logger.Fatal(ErrReloadNginx, zap.String("error", err.Error()))
				}

				logger.Info(SuccessNginxReload)
			} else {
				logger.Info("node not found in the cluster.workers, skipping remove operation", zap.String("node", worker.HostIP))
			}
		},
	})
	informerFactory.Start(wait.NeverStop)
	informerFactory.WaitForCacheSync(wait.NeverStop)
}
