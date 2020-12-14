package main

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"log"
	// "k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	// "os"
	"time"
	_ "time"
)

func runNodeInformer(cluster *Cluster, clientSet *kubernetes.Clientset) {
	informerFactory := informers.NewSharedInformerFactory(clientSet, time.Second * 30)
	nodeInformer := informerFactory.Core().V1().Nodes()
	nodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			node := obj.(*v1.Node)
			_, ok := node.Labels[*workerNodeLabel]
			nodeReady := isNodeReady(node)

			if ok && nodeReady == "True" {
				log.Printf("adding node %v to the cluster.Workers\n", node.Status.Addresses[0].Address)
				worker := newWorker(cluster.MasterIP, node.Status.Addresses[0].Address, nodeReady)

				// add worker to cluster.Workers slice
				addWorker(&cluster.Workers, worker)
				log.Printf("final cluster.Workers = %v\n", cluster.Workers)

				// add worker to each NodePort.Workers in the cluster.NodePorts slice
				addWorkerToNodePorts(cluster.NodePorts, worker)

				// Apply changes to the template
				renderTemplate(*templateInputFile, *templateOutputFile, nginxConf)

				// err = reloadNginx()
				// checkError(err)
			}
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			oldNode := oldObj.(*v1.Node)
			newNode := newObj.(*v1.Node)

			// there is an update for sure
			if oldNode.ResourceVersion != newNode.ResourceVersion {
				_, newOk := newNode.Labels[*workerNodeLabel]
				oldWorker := newWorker(cluster.MasterIP, oldNode.Status.Addresses[0].Address, isNodeReady(oldNode))
				newWorker := newWorker(cluster.MasterIP, newNode.Status.Addresses[0].Address, isNodeReady(newNode))

				oldWorkerIndex, oldWorkerFound := findWorker(cluster.Workers, *oldWorker)
				if oldWorkerFound {
					// - old node was at the slice cluster.Workers:
					//   - new node is no more healthy or new node is no more labeled
					//     - remove node from cluster.Workers
					//   - new node is still healthy and labelled
					//     - do nothing
					if newWorker.NodeCondition == "True" && newOk {
						log.Printf("node %v is still healthy and labelled, skipping...\n", *oldWorker)
					} else {
						log.Printf("node %v is not healthy or is not labelled, removing from cluster.Workers!\n",
							*oldWorker)
						removeWorker(&cluster.Workers, oldWorkerIndex)
						log.Printf("final cluster.Workers = %v\n", cluster.Workers)

						removeWorkerFromNodePorts(&cluster.NodePorts, oldWorker)
						log.Printf("successfully removed worker %v from each NodePort in the cluster.NodePorts\n", oldWorker)

						// Apply changes to the template
						renderTemplate(*templateInputFile, *templateOutputFile, nginxConf)

						// err = reloadNginx()
						// checkError(err)
					}
				} else {
					// - old node was not at the slice cluster.Workers:
					//   - new node is healthy and node is labelled
					//     - ensure old node was not at cluster.Workers, add new node to cluster.Workers.
					//   - new node is not healthy or not labelled
					if newWorker.NodeCondition == "True" && newOk {
						// add newWorker to cluster.Workers slice
						log.Printf("node %v is now healthy and labeled, adding to the cluster.Workers slice...\n",
							*oldWorker)
						addWorker(&cluster.Workers, newWorker)
						log.Printf("final cluster.Workers = %v\n", cluster.Workers)

						// add newWorker to each NodePort.Workers in the cluster.NodePorts slice
						addWorkerToNodePorts(cluster.NodePorts, newWorker)

						// Apply changes to the template
						renderTemplate(*templateInputFile, *templateOutputFile, nginxConf)

						// err = reloadNginx()
						// checkError(err)
					} else {
						log.Printf("node %v is still unhealthy or unlabelled, skipping...\n", *oldWorker)
					}
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			node := obj.(*v1.Node)
			nodeReady := isNodeReady(node)
			worker := newWorker(cluster.MasterIP, node.Status.Addresses[0].Address, nodeReady)
			log.Printf("delete event fetched for worker %v!\n", worker)
			index, found := findWorker(cluster.Workers, *worker)
			if found {
				log.Printf("worker %v found in the cluster.Workers, removing...\n", worker)
				removeWorker(&cluster.Workers, index)
				log.Printf("successfully removed worker %v from cluster.Workers slice!\n", worker)
				log.Printf("final cluster.Workers after delete operation = %v\n", cluster.Workers)

				removeWorkerFromNodePorts(&cluster.NodePorts, worker)
				log.Printf("successfully removed worker %v from each NodePort in the cluster.NodePorts\n", worker)
				log.Printf("final cluster.NodePorts after delete operation = %v\n", cluster.NodePorts)

				// Apply changes to the template
				renderTemplate(*templateInputFile, *templateOutputFile, nginxConf)

				// err = reloadNginx()
				// checkError(err)
			} else {
				log.Printf("worker %v NOT found in the cluster.Workers, skipping remove operation!\n", worker)
			}
		},
	})
	informerFactory.Start(wait.NeverStop)
	informerFactory.WaitForCacheSync(wait.NeverStop)
}

func runServiceInformer(cluster *Cluster, clientSet *kubernetes.Clientset) {
	informerFactory := informers.NewSharedInformerFactory(clientSet, time.Second * 30)
	serviceInformer := informerFactory.Core().V1().Services()
	serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			service := obj.(*v1.Service)
			_, ok := service.Annotations[*customAnnotation]
			if service.Spec.Type == "NodePort" && ok {
				log.Printf("service %v is added on namespace %v with nodeport %v!\n", service.Name, service.Namespace,
					service.Spec.Ports[0].NodePort)

				nodePort := newNodePort(cluster.MasterIP, service.Spec.Ports[0].NodePort)
				index, found := findNodePort(cluster.NodePorts, *nodePort)
				if found {
					nodePort = cluster.NodePorts[index]
					log.Printf("NodePort %v found in the backend.NodePorts, skipping adding...\n", nodePort)
				} else {
					addNodePort(&cluster.NodePorts, nodePort)
				}

				// TODO: Add cluster.Workers to newly created NodePort and render the template again (TEST)
				addWorkersToNodePort(cluster.Workers, nodePort)

				// Apply changes to the template
				renderTemplate(*templateInputFile, *templateOutputFile, nginxConf)

				// err = reloadNginx()
				// checkError(err)
			} else {
				log.Printf("service %v on namespace %v is not annotated or NodePort type!\n", service.Name,
					service.Spec.Type)
			}
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			oldService := oldObj.(*v1.Service)
			newService := newObj.(*v1.Service)
			// There is an actual update on the service
			if oldService.ResourceVersion != newService.ResourceVersion {
				/*
				- Old service was labelled and NodePort type:
					- check if new service also labelled and NodePort type
						- if yes, check if ports are the same
							- if yes, do nothing
							- if no, update old service with the new service
						- if no, remove old service from slice
				*/
				log.Printf("service %v is updated on namespace %v!\n", oldService.Name, oldService.Namespace)
				_, oldOk := oldService.Annotations[*customAnnotation]
				_, newOk := newService.Annotations[*customAnnotation]
				if oldOk && oldService.Spec.Type == "NodePort" {
					if newOk && newService.Spec.Type == "NodePort" {
						oldNodePort := newNodePort(cluster.MasterIP, oldService.Spec.Ports[0].NodePort)
						newNodePort := newNodePort(cluster.MasterIP, newService.Spec.Ports[0].NodePort)
						if !oldNodePort.Equals(newNodePort) {
							updateNodePort(&cluster.NodePorts, cluster.Workers, oldNodePort, newNodePort)
							log.Printf("final cluster.NodePorts after update operation = %v\n", cluster.NodePorts)
						} else {
							log.Println("NodePort objects are the same on the updated service, nothing to do!")
						}
					} else {
						oldNodePort := newNodePort(cluster.MasterIP, oldService.Spec.Ports[0].NodePort)
						oldIndex, oldFound := findNodePort(cluster.NodePorts, *oldNodePort)
						if oldFound {
							log.Printf("removing service %v from cluster.NodePorts because it is no more " +
								"NodePort type or labelled!\n", oldService.Name)
							removeNodePort(&cluster.NodePorts, oldIndex)
							log.Printf("final cluster.NodePorts after delete operation = %v\n", cluster.NodePorts)
						}
					}
				} else {
					/*
					- Old service was not labelled or not a NodePort type service:
						- ensure that slice does not contain that old service
							- if yes, remove old service from slice
							- if no, do nothing
						- check if new service labelled and NodePort type:
							- if yes, add to the slice
							- if no, do nothing
					*/
					oldNodePort := newNodePort(cluster.MasterIP, oldService.Spec.Ports[0].NodePort)
					oldIndex, oldFound := findNodePort(cluster.NodePorts, *oldNodePort)
					if oldFound {
						log.Printf("removing service %v from cluster.NodePorts because it is accidentally added!\n", oldService.Name)
						removeNodePort(&cluster.NodePorts, oldIndex)
						log.Printf("final cluster.NodePorts after delete operation = %v\n", cluster.NodePorts)
					}

					if newOk && newService.Spec.Type == "NodePort" {
						newNodePort := newNodePort(cluster.MasterIP, newService.Spec.Ports[0].NodePort)
						_, newFound := findNodePort(cluster.NodePorts, *newNodePort)
						if !newFound {
							log.Printf("adding service %v to cluster.NodePorts because it is labelled and NodePort type!\n",
								newService.Name)
							addNodePort(&cluster.NodePorts, newNodePort)
							// TODO: Add cluster.Workers to newly created NodePort (TEST)
							addWorkersToNodePort(cluster.Workers, newNodePort)
						} else {
							log.Printf("service %v already found in cluster.NodePorts, this is buggy, inspect! " +
								"skipping adding operation!\n", newService.Name)
						}

					}
				}

				// Apply changes to the template
				renderTemplate(*templateInputFile, *templateOutputFile, nginxConf)

				// err = reloadNginx()
				// checkError(err)
			}
		},
		DeleteFunc: func(obj interface{}) {
			/*
			- Check if deleted service was labelled and NodePort type
				- If yes, remove it from slice
				- If no, do nothing
			 */

			service := obj.(*v1.Service)
			_, ok := service.Annotations[*customAnnotation]
			if service.Spec.Type == "NodePort" && ok {
				log.Printf("service %v is deleted on namespace %v!\n", service.Name, service.Namespace)
				nodePort := newNodePort(cluster.MasterIP, service.Spec.Ports[0].NodePort)
				index, found := findNodePort(cluster.NodePorts, *nodePort)
				if found {
					log.Printf("deleted service %v found on the cluster.NodePorts slice, removing!\n", service.Name)
					removeNodePort(&cluster.NodePorts, index)
					log.Printf("final cluster.NodePorts after delete operation = %v\n", cluster.NodePorts)
				}
			}

			// Apply changes to the template
			renderTemplate(*templateInputFile, *templateOutputFile, nginxConf)

			// err = reloadNginx()
			// checkError(err)
		},
	})
	informerFactory.Start(wait.NeverStop)
	informerFactory.WaitForCacheSync(wait.NeverStop)
}