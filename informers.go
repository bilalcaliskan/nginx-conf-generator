package main

import (
	"fmt"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"time"
	_ "time"
)

func runNodeInformer(cluster *Cluster, clientSet *kubernetes.Clientset, logger *zap.Logger) {
	defer func() {
		err = logger.Sync()
		if err != nil {
			panic(err)
		}
	}()

	informerFactory := informers.NewSharedInformerFactory(clientSet, time.Second * 30)
	nodeInformer := informerFactory.Core().V1().Nodes()
	nodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			node := obj.(*v1.Node)
			_, ok := node.Labels[workerNodeLabel]
			nodeReady := isNodeReady(node)

			if ok && nodeReady == "True" {
				logger.Info("adding node to the cluster.Workers", zap.String("node", node.Status.Addresses[0].Address))
				worker := newWorker(cluster.MasterIP, node.Status.Addresses[0].Address, nodeReady)

				// add worker to cluster.Workers slice
				addWorker(&cluster.Workers, worker)
				logger.Debug(fmt.Sprintf("final cluster.Workers after add operation: %v\n", cluster.Workers))

				// add worker to each NodePort.Workers in the cluster.NodePorts slice
				addWorkerToNodePorts(cluster.NodePorts, worker)

				// Apply changes to the template
				renderTemplate(templateInputFile, templateOutputFile, nginxConf)

				// reload Nginx service
				reloadNginx(logger)
			}
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			oldNode := oldObj.(*v1.Node)
			newNode := newObj.(*v1.Node)

			// there is an update for sure
			if oldNode.ResourceVersion != newNode.ResourceVersion {
				_, newOk := newNode.Labels[workerNodeLabel]
				oldWorker := newWorker(cluster.MasterIP, oldNode.Status.Addresses[0].Address, isNodeReady(oldNode))
				newWorker := newWorker(cluster.MasterIP, newNode.Status.Addresses[0].Address, isNodeReady(newNode))

				oldWorkerIndex, oldWorkerFound := findWorker(cluster.Workers, *oldWorker)
				if oldWorkerFound {
					if newWorker.NodeCondition == "True" && newOk {
						logger.Info("node is still healthy and labelled, skipping...", zap.Any("node", *oldWorker))
					} else {
						logger.Info("node is not healthy or is not labelled, removing from cluster.Workers!",
							zap.String("node", oldWorker.HostIP))
						removeWorker(&cluster.Workers, oldWorkerIndex)
						logger.Debug(fmt.Sprintf("final cluster.Workers after delete operation: %v\n", cluster.Workers))

						removeWorkerFromNodePorts(&cluster.NodePorts, oldWorker)
						logger.Info("successfully removed node from each NodePort in the cluster.NodePorts",
							zap.String("node", oldWorker.HostIP))

						// Apply changes to the template
						renderTemplate(templateInputFile, templateOutputFile, nginxConf)

						// reload Nginx service
						reloadNginx(logger)
					}
				} else {
					if newWorker.NodeCondition == "True" && newOk {
						logger.Info("node is now healthy and labeled, adding to the cluster.Workers slice",
							zap.String("node", oldWorker.HostIP))
						addWorker(&cluster.Workers, newWorker)
						logger.Debug(fmt.Sprintf("final cluster.Workers after add operation: %v\n", cluster.Workers))

						// add newWorker to each NodePort.Workers in the cluster.NodePorts slice
						addWorkerToNodePorts(cluster.NodePorts, newWorker)

						// Apply changes to the template
						renderTemplate(templateInputFile, templateOutputFile, nginxConf)

						// reload Nginx service
						reloadNginx(logger)
					} else {
						logger.Info("node is still unhealthy or unlabelled, skipping...", zap.String("node",
							oldWorker.HostIP))
					}
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			node := obj.(*v1.Node)
			nodeReady := isNodeReady(node)
			worker := newWorker(cluster.MasterIP, node.Status.Addresses[0].Address, nodeReady)
			logger.Info("delete event fetched for node", zap.String("node", worker.HostIP))
			index, found := findWorker(cluster.Workers, *worker)
			if found {
				logger.Info("node found in the cluster.Workers, removing...", zap.String("node", worker.HostIP))
				removeWorker(&cluster.Workers, index)
				logger.Info("successfully removed node from cluster.Workers slice!", zap.String("node",
					worker.HostIP))
				logger.Debug(fmt.Sprintf("final cluster.Workers = %v\n", cluster.Workers))

				removeWorkerFromNodePorts(&cluster.NodePorts, worker)
				logger.Info("successfully removed node from each NodePort in the cluster.NodePorts",
					zap.String("node", worker.HostIP))
				logger.Debug(fmt.Sprintf("final cluster.NodePorts after delete operation: %v\n", cluster.NodePorts))

				// Apply changes to the template
				renderTemplate(templateInputFile, templateOutputFile, nginxConf)

				// reload Nginx service
				reloadNginx(logger)
			} else {
				logger.Info("node not found in the cluster.workers, skipping remove operation",
					zap.String("node", worker.HostIP))
			}
		},
	})
	informerFactory.Start(wait.NeverStop)
	informerFactory.WaitForCacheSync(wait.NeverStop)
}

func runServiceInformer(cluster *Cluster, clientSet *kubernetes.Clientset, logger *zap.Logger) {
	defer func() {
		err = logger.Sync()
		if err != nil {
			panic(err)
		}
	}()

	informerFactory := informers.NewSharedInformerFactory(clientSet, time.Second * 30)
	serviceInformer := informerFactory.Core().V1().Services()
	serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			service := obj.(*v1.Service)
			_, ok := service.Annotations[customAnnotation]
			if service.Spec.Type == "NodePort" && ok {
				logger.Info("valid service added", zap.String("name", service.Name),
					zap.String("namespace", service.Namespace), zap.Int32("nodePort", service.Spec.Ports[0].NodePort))

				nodePort := newNodePort(cluster.MasterIP, service.Spec.Ports[0].NodePort)
				index, found := findNodePort(cluster.NodePorts, *nodePort)
				if found {
					nodePort = cluster.NodePorts[index]
					logger.Info("NodePort found in the backend.NodePorts, skipping adding...",
						zap.Int32("nodePort", nodePort.Port))
				} else {
					addNodePort(&cluster.NodePorts, nodePort)
				}

				addWorkersToNodePort(cluster.Workers, nodePort)

				// Apply changes to the template
				renderTemplate(templateInputFile, templateOutputFile, nginxConf)

				// Reload Nginx service
				reloadNginx(logger)
			} else {
				logger.Info("service is not valid, it is not annotated or not a NodePort type service",
					zap.String("name", service.Name), zap.String("namespace", service.Namespace))
			}
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			oldService := oldObj.(*v1.Service)
			newService := newObj.(*v1.Service)
			// There is an actual update on the service
			if oldService.ResourceVersion != newService.ResourceVersion {
				_, oldOk := oldService.Annotations[customAnnotation]
				_, newOk := newService.Annotations[customAnnotation]
				if oldOk && oldService.Spec.Type == "NodePort" {
					if newOk && newService.Spec.Type == "NodePort" {
						oldNodePort := newNodePort(cluster.MasterIP, oldService.Spec.Ports[0].NodePort)
						newNodePort := newNodePort(cluster.MasterIP, newService.Spec.Ports[0].NodePort)
						if !oldNodePort.Equals(newNodePort) {
							logger.Info("NodePort changed on the valid service, updating",
								zap.String("name", oldService.Name), zap.String("namespace", oldService.Namespace),
								zap.Int32("oldNodePort", oldNodePort.Port), zap.Int32("newNodePort", newNodePort.Port))
							updateNodePort(&cluster.NodePorts, cluster.Workers, oldNodePort, newNodePort)
							logger.Debug(fmt.Sprintf("final cluster.NodePorts after update operation: %v\n", cluster.NodePorts))
						} else {
							logger.Info("NodePort ports are the same on the updated service, nothing to do",
								zap.String("name", oldService.Name), zap.String("namespace", oldService.Namespace),
								zap.Int32("oldNodePort", oldNodePort.Port), zap.Int32("newNodePort", newNodePort.Port))
						}
					} else {
						oldNodePort := newNodePort(cluster.MasterIP, oldService.Spec.Ports[0].NodePort)
						oldIndex, oldFound := findNodePort(cluster.NodePorts, *oldNodePort)
						if oldFound {
							logger.Info("removing service from cluster.NodePorts because it is no more NodePort " +
								"type or no more labelled!", zap.String("name", oldService.Name),
								zap.String("namespace", oldService.Namespace))
							removeNodePort(&cluster.NodePorts, oldIndex)
							logger.Info("successfully removed service from cluster.NodePorts",
								zap.String("name", oldService.Name), zap.String("namespace", oldService.Namespace))
							logger.Debug(fmt.Sprintf("final cluster.NodePorts after delete operation: %v\n", cluster.NodePorts))
						}
					}
				} else {
					oldNodePort := newNodePort(cluster.MasterIP, oldService.Spec.Ports[0].NodePort)
					oldIndex, oldFound := findNodePort(cluster.NodePorts, *oldNodePort)
					if oldFound {
						logger.Info("removing service from cluster.NodePorts because it is accidentially added",
							zap.String("name", oldService.Name), zap.String("namespace", oldService.Namespace))
						removeNodePort(&cluster.NodePorts, oldIndex)
						logger.Info("successfully removed service from cluster.NodePorts",
							zap.String("name", oldService.Name), zap.String("namespace", oldService.Namespace))
						logger.Debug(fmt.Sprintf("final cluster.NodePorts after delete operation: %v\n", cluster.NodePorts))
					}

					if newOk && newService.Spec.Type == "NodePort" {
						newNodePort := newNodePort(cluster.MasterIP, newService.Spec.Ports[0].NodePort)
						_, newFound := findNodePort(cluster.NodePorts, *newNodePort)
						if !newFound {
							logger.Info("adding service to cluster.NodePorts because it is labelled and NodePort " +
								"type service", zap.String("name", newService.Name), zap.String("namespace", newService.Namespace),
								zap.Int32("nodePort", newNodePort.Port))
							addNodePort(&cluster.NodePorts, newNodePort)
							addWorkersToNodePort(cluster.Workers, newNodePort)
							logger.Info("successfully added service to cluster.NodePorts",
								zap.String("name", newService.Name),
								zap.String("namespace", newService.Namespace),
								zap.Int32("nodePort", newNodePort.Port))
						} else {
							logger.Warn("service is already found in cluster.NodePorts, this is buggy, inspect!",
								zap.String("name", newService.Name),
								zap.String("namespace", newService.Namespace))
						}

					}
				}

				// Apply changes to the template
				renderTemplate(templateInputFile, templateOutputFile, nginxConf)

				// Reload Nginx service
				reloadNginx(logger)
			}
		},
		DeleteFunc: func(obj interface{}) {
			service := obj.(*v1.Service)
			_, ok := service.Annotations[customAnnotation]
			if service.Spec.Type == "NodePort" && ok {
				nodePort := newNodePort(cluster.MasterIP, service.Spec.Ports[0].NodePort)
				index, found := findNodePort(cluster.NodePorts, *nodePort)
				if found {
					logger.Info("valid service deleted, removing from cluster.NodePorts",
						zap.String("name", service.Name),
						zap.String("namespace", service.Namespace))
					removeNodePort(&cluster.NodePorts, index)
					logger.Info("successfully removed deleted service from cluster.NodePorts",
						zap.String("name", service.Name),
						zap.String("namespace", service.Namespace))
					logger.Debug(fmt.Sprintf("final cluster.NodePorts after delete operation: %v\n", cluster.NodePorts))
				}
			}

			// Apply changes to the template
			renderTemplate(templateInputFile, templateOutputFile, nginxConf)

			// Reload Nginx service
			reloadNginx(logger)
		},
	})
	informerFactory.Start(wait.NeverStop)
	informerFactory.WaitForCacheSync(wait.NeverStop)
}