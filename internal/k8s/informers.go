package k8s

import (
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"nginx-conf-generator/internal/logging"
	"nginx-conf-generator/internal/metrics"
	"nginx-conf-generator/internal/options"
	"time"
)

var (
	logger *zap.Logger
)

func init() {
	logger = logging.GetLogger()
}

// RunNodeInformer spins up a shared informer factory and fetch Kubernetes node events
func RunNodeInformer(cluster *Cluster, clientSet *kubernetes.Clientset, ncgo *options.NginxConfGeneratorOptions,
	nginxConf *NginxConf) {
	informerFactory := informers.NewSharedInformerFactory(clientSet, time.Second*30)
	nodeInformer := informerFactory.Core().V1().Nodes()
	nodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			node := obj.(*v1.Node)
			_, ok := node.Labels[ncgo.WorkerNodeLabel]
			nodeReady := isNodeReady(node)

			if ok && nodeReady == "True" {
				logger.Info("adding node to the cluster.Workers",
					zap.String("masterIP", cluster.MasterIP), zap.String("node", node.Status.Addresses[0].Address))
				worker := NewWorker(cluster.MasterIP, node.Status.Addresses[0].Address, nodeReady)

				// add Worker to cluster.Workers slice
				addWorker(&cluster.Workers, worker)
				metrics.TargetNodeCounter.Add(1)

				// add Worker to each nodePort.Workers in the cluster.NodePorts slice
				addWorkerToNodePorts(cluster.NodePorts, worker)

				// Apply changes to the template
				err := renderTemplate(ncgo.TemplateInputFile, ncgo.TemplateOutputFile, nginxConf)
				if err != nil {
					logger.Fatal(ErrRenderTemplate, zap.Error(err))
				}

				// reload Nginx service
				err = reloadNginx()
				if err != nil {
					logger.Fatal(ErrReloadNginx, zap.String("error", err.Error()))
				} else {
					logger.Info(SuccessNginxReload)
				}
			}
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			oldNode := oldObj.(*v1.Node)
			newNode := newObj.(*v1.Node)

			// there is an update for sure
			if oldNode.ResourceVersion != newNode.ResourceVersion {
				_, newOk := newNode.Labels[ncgo.WorkerNodeLabel]
				oldWorker := NewWorker(cluster.MasterIP, oldNode.Status.Addresses[0].Address, isNodeReady(oldNode))
				newWorker := NewWorker(cluster.MasterIP, newNode.Status.Addresses[0].Address, isNodeReady(newNode))

				oldWorkerIndex, oldWorkerFound := findWorker(cluster.Workers, *oldWorker)
				if oldWorkerFound {
					if newWorker.NodeCondition == "True" && newOk {
						logger.Info("node is still healthy and labelled, skipping...",
							zap.String("masterIP", cluster.MasterIP), zap.String("node", oldWorker.HostIP))
					} else {
						logger.Info("node is not healthy or is not labelled, removing from cluster.Workers!",
							zap.String("masterIP", cluster.MasterIP), zap.String("node", oldWorker.HostIP))
						removeWorker(&cluster.Workers, oldWorkerIndex)

						removeWorkerFromNodePorts(&cluster.NodePorts, oldWorker)
						logger.Info("successfully removed node from each nodePort in the cluster.NodePorts",
							zap.String("masterIP", cluster.MasterIP), zap.String("node", oldWorker.HostIP))
						metrics.TargetNodeCounter.Add(-1)

						// Apply changes to the template
						err := renderTemplate(ncgo.TemplateInputFile, ncgo.TemplateOutputFile, nginxConf)
						if err != nil {
							logger.Fatal(ErrRenderTemplate, zap.Error(err))
						}

						// reload Nginx service
						err = reloadNginx()
						if err != nil {
							logger.Fatal(ErrReloadNginx, zap.String("error", err.Error()))
						} else {
							logger.Info(SuccessNginxReload)
						}
					}
				} else {
					if newWorker.NodeCondition == "True" && newOk {
						logger.Info("node is now healthy and labeled, adding to the cluster.Workers slice",
							zap.String("masterIP", cluster.MasterIP), zap.String("node", oldWorker.HostIP))
						addWorker(&cluster.Workers, newWorker)

						// add NewWorker to each nodePort.Workers in the cluster.NodePorts slice
						addWorkerToNodePorts(cluster.NodePorts, newWorker)
						metrics.TargetNodeCounter.Add(1)

						// Apply changes to the template
						err := renderTemplate(ncgo.TemplateInputFile, ncgo.TemplateOutputFile, nginxConf)
						if err != nil {
							logger.Fatal(ErrRenderTemplate, zap.Error(err))
						}

						// reload Nginx service
						err = reloadNginx()
						if err != nil {
							logger.Fatal(ErrReloadNginx, zap.String("error", err.Error()))
						} else {
							logger.Info(SuccessNginxReload)
						}
					} else {
						logger.Info("node is still unhealthy or unlabelled, skipping...",
							zap.String("masterIP", cluster.MasterIP), zap.String("node", oldWorker.HostIP))
					}
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			node := obj.(*v1.Node)
			nodeReady := isNodeReady(node)
			worker := NewWorker(cluster.MasterIP, node.Status.Addresses[0].Address, nodeReady)
			logger.Info("delete event fetched for node", zap.String("masterIP", cluster.MasterIP),
				zap.String("node", worker.HostIP))
			index, found := findWorker(cluster.Workers, *worker)
			if found {
				logger.Info("node found in the cluster.Workers, removing...",
					zap.String("masterIP", cluster.MasterIP), zap.String("node", worker.HostIP))
				removeWorker(&cluster.Workers, index)
				logger.Info("successfully removed node from cluster.Workers slice!",
					zap.String("masterIP", cluster.MasterIP), zap.String("node", worker.HostIP))

				removeWorkerFromNodePorts(&cluster.NodePorts, worker)
				logger.Info("successfully removed node from each nodePort in the cluster.NodePorts",
					zap.String("masterIP", cluster.MasterIP), zap.String("node", worker.HostIP))
				metrics.TargetNodeCounter.Add(-1)

				// Apply changes to the template
				err := renderTemplate(ncgo.TemplateInputFile, ncgo.TemplateOutputFile, nginxConf)
				if err != nil {
					logger.Fatal(ErrRenderTemplate, zap.Error(err))
				}

				// reload Nginx service
				err = reloadNginx()
				if err != nil {
					logger.Fatal(ErrReloadNginx, zap.String("error", err.Error()))
				} else {
					logger.Info(SuccessNginxReload)
				}
			} else {
				logger.Info("node not found in the cluster.workers, skipping remove operation",
					zap.String("masterIP", cluster.MasterIP), zap.String("node", worker.HostIP))
			}
		},
	})
	informerFactory.Start(wait.NeverStop)
	informerFactory.WaitForCacheSync(wait.NeverStop)
}

// RunServiceInformer spins up a shared informer factory and fetch Kubernetes service events
func RunServiceInformer(cluster *Cluster, clientSet *kubernetes.Clientset, ncgo *options.NginxConfGeneratorOptions,
	nginxConf *NginxConf) {
	informerFactory := informers.NewSharedInformerFactory(clientSet, time.Second*30)
	serviceInformer := informerFactory.Core().V1().Services()
	serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			service := obj.(*v1.Service)
			var ok bool

			if service.Annotations[ncgo.CustomAnnotation] == "true" {
				ok = true
			}

			if service.Spec.Type == "NodePort" && ok && service.Annotations[ncgo.CustomAnnotation] == "true" {
				logger.Info("valid service added", zap.String("name", service.Name),
					zap.String("masterIP", cluster.MasterIP), zap.String("namespace", service.Namespace),
					zap.Int32("nodePort", service.Spec.Ports[0].NodePort))

				nodePort := newNodePort(cluster.MasterIP, service.Spec.Ports[0].NodePort)
				index, found := findNodePort(cluster.NodePorts, *nodePort)
				if found {
					nodePort = cluster.NodePorts[index]
					logger.Info("nodePort found in the backend.NodePorts, skipping adding...",
						zap.String("masterIP", cluster.MasterIP), zap.Int32("nodePort", nodePort.Port))
				} else {
					addNodePort(&cluster.NodePorts, nodePort)
				}

				addWorkersToNodePort(cluster.Workers, nodePort)

				// Apply changes to the template
				err := renderTemplate(ncgo.TemplateInputFile, ncgo.TemplateOutputFile, nginxConf)
				if err != nil {
					logger.Fatal(ErrRenderTemplate, zap.Error(err))
				}

				// reload Nginx service
				err = reloadNginx()
				if err != nil {
					logger.Fatal(ErrReloadNginx, zap.String("error", err.Error()))
				} else {
					logger.Info(SuccessNginxReload)
				}
			} else {
				logger.Info("service is not valid, it is not annotated or not a nodePort type service",
					zap.String("masterIP", cluster.MasterIP), zap.String("name", service.Name),
					zap.String("namespace", service.Namespace))
			}
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			oldService := oldObj.(*v1.Service)
			newService := newObj.(*v1.Service)
			// There is an actual update on the service
			if oldService.ResourceVersion != newService.ResourceVersion {
				var oldOk, newOk bool
				if oldService.Annotations[ncgo.CustomAnnotation] == "true" {
					oldOk = true
				}

				if newService.Annotations[ncgo.CustomAnnotation] == "true" {
					newOk = true
				}

				// _, newOk := (newService.Annotations[ncgo.CustomAnnotation] && newService.Annotations[ncgo.CustomAnnotation] == "true")
				if oldOk && oldService.Spec.Type == "NodePort" {
					if newOk && newService.Spec.Type == "NodePort" {
						oldNodePort := newNodePort(cluster.MasterIP, oldService.Spec.Ports[0].NodePort)
						newNodePort := newNodePort(cluster.MasterIP, newService.Spec.Ports[0].NodePort)
						if !oldNodePort.Equals(newNodePort) {
							logger.Info("nodePort changed on the valid service, updating",
								zap.String("masterIP", cluster.MasterIP), zap.String("name", oldService.Name),
								zap.String("namespace", oldService.Namespace), zap.Int32("oldNodePort", oldNodePort.Port),
								zap.Int32("newNodePort", newNodePort.Port))
							updateNodePort(&cluster.NodePorts, cluster.Workers, oldNodePort, newNodePort)
						} else {
							logger.Info("nodePort ports are the same on the updated service, nothing to do",
								zap.String("masterIP", cluster.MasterIP), zap.String("name", oldService.Name),
								zap.String("namespace", oldService.Namespace), zap.Int32("oldNodePort", oldNodePort.Port),
								zap.Int32("newNodePort", newNodePort.Port))
						}
					} else {
						oldNodePort := newNodePort(cluster.MasterIP, oldService.Spec.Ports[0].NodePort)
						oldIndex, oldFound := findNodePort(cluster.NodePorts, *oldNodePort)
						if oldFound {
							logger.Info("removing service from cluster.NodePorts because it is no more nodePort "+
								"type or no more labelled!", zap.String("masterIP", cluster.MasterIP),
								zap.String("name", oldService.Name), zap.String("namespace", oldService.Namespace))
							removeNodePort(&cluster.NodePorts, oldIndex)
							logger.Info("successfully removed service from cluster.NodePorts",
								zap.String("masterIP", cluster.MasterIP), zap.String("name", oldService.Name),
								zap.String("namespace", oldService.Namespace))
						}
					}
				} else {
					oldNodePort := newNodePort(cluster.MasterIP, oldService.Spec.Ports[0].NodePort)
					oldIndex, oldFound := findNodePort(cluster.NodePorts, *oldNodePort)
					if oldFound {
						logger.Info("removing service from cluster.NodePorts because it is accidentially added",
							zap.String("masterIP", cluster.MasterIP), zap.String("name", oldService.Name),
							zap.String("namespace", oldService.Namespace))
						removeNodePort(&cluster.NodePorts, oldIndex)
						logger.Info("successfully removed service from cluster.NodePorts",
							zap.String("masterIP", cluster.MasterIP), zap.String("name", oldService.Name),
							zap.String("namespace", oldService.Namespace))
					}

					if newOk && newService.Spec.Type == "NodePort" {
						newNodePort := newNodePort(cluster.MasterIP, newService.Spec.Ports[0].NodePort)
						_, newFound := findNodePort(cluster.NodePorts, *newNodePort)
						if !newFound {
							logger.Info("adding service to cluster.NodePorts because it is labelled and nodePort "+
								"type service",
								zap.String("masterIP", cluster.MasterIP), zap.String("name", newService.Name),
								zap.String("namespace", newService.Namespace), zap.Int32("nodePort", newNodePort.Port))
							addNodePort(&cluster.NodePorts, newNodePort)
							addWorkersToNodePort(cluster.Workers, newNodePort)
							logger.Info("successfully added service to cluster.NodePorts",
								zap.String("masterIP", cluster.MasterIP), zap.String("name", newService.Name),
								zap.String("namespace", newService.Namespace), zap.Int32("nodePort", newNodePort.Port))
						} else {
							logger.Warn("service is already found in cluster.NodePorts, this is buggy, inspect!",
								zap.String("masterIP", cluster.MasterIP), zap.String("name", newService.Name),
								zap.String("namespace", newService.Namespace))
						}

					}
				}

				// Apply changes to the template
				err := renderTemplate(ncgo.TemplateInputFile, ncgo.TemplateOutputFile, nginxConf)
				if err != nil {
					logger.Fatal(ErrRenderTemplate, zap.Error(err))
				}

				// reload Nginx service
				err = reloadNginx()
				if err != nil {
					logger.Fatal(ErrReloadNginx, zap.String("error", err.Error()))
				} else {
					logger.Info(SuccessNginxReload)
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			service := obj.(*v1.Service)
			var ok bool
			if service.Annotations[ncgo.CustomAnnotation] == "true" {
				ok = true
			}

			if service.Spec.Type == "NodePort" && ok {
				nodePort := newNodePort(cluster.MasterIP, service.Spec.Ports[0].NodePort)
				index, found := findNodePort(cluster.NodePorts, *nodePort)
				if found {
					logger.Info("valid service deleted, removing from cluster.NodePorts",
						zap.String("masterIP", cluster.MasterIP), zap.String("name", service.Name),
						zap.String("namespace", service.Namespace))
					removeNodePort(&cluster.NodePorts, index)
					logger.Info("successfully removed deleted service from cluster.NodePorts",
						zap.String("masterIP", cluster.MasterIP), zap.String("name", service.Name),
						zap.String("namespace", service.Namespace))
				}
			}

			// Apply changes to the template
			err := renderTemplate(ncgo.TemplateInputFile, ncgo.TemplateOutputFile, nginxConf)
			if err != nil {
				logger.Fatal(ErrRenderTemplate, zap.Error(err))
			}

			// reload Nginx service
			err = reloadNginx()
			if err != nil {
				logger.Fatal(ErrReloadNginx, zap.String("error", err.Error()))
			} else {
				logger.Info(SuccessNginxReload)
			}
		},
	})
	informerFactory.Start(wait.NeverStop)
	informerFactory.WaitForCacheSync(wait.NeverStop)
}
