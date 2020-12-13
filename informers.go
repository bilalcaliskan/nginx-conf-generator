package main

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"log"
	"os"
	"text/template"

	// "k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	// "os"
	"time"
	_ "time"
)

func runNodeInformer(cluster *Cluster, clientSet *kubernetes.Clientset, workerNodeLabel string) {
	informerFactory := informers.NewSharedInformerFactory(clientSet, time.Second * 30)
	nodeInformer := informerFactory.Core().V1().Nodes()
	nodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			node := obj.(*v1.Node)
			_, ok := node.Labels[workerNodeLabel]
			nodeReady := isNodeReady(node)

			if ok && nodeReady == "True" {
				log.Printf("adding node %v to the cluster.Workers\n", node.Status.Addresses[0].Address)
				worker := newWorker(cluster.MasterIP, node.Status.Addresses[0].Address, nodeReady)
				cluster.Workers = append(cluster.Workers, worker)
				log.Printf("final cluster.Workers = %v\n", cluster.Workers)
			}

			go runServiceInformer(cluster, clientSet, *customAnnotation, *templateInputFile, *templateOutputFile)
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
						cluster.Workers = removeWorker(cluster.Workers, oldWorkerIndex)
						log.Printf("final cluster.Workers = %v\n", cluster.Workers)
					}
				} else {
					// - old node was not at the slice cluster.Workers:
					//   - new node is healthy and node is labelled
					//     - ensure old node was not at cluster.Workers, add new node to cluster.Workers.
					//   - new node is not healthy or not labelled
					if newWorker.NodeCondition == "True" && newOk {
						log.Printf("node %v is now healthy and labeled, adding to the cluster.Workers slice...\n",
							*oldWorker)
						cluster.Workers = append(cluster.Workers, newWorker)
						log.Printf("final cluster.Workers = %v\n", cluster.Workers)
					} else {
						log.Printf("node %v is still unhealthy or unlabelled, skipping...\n", *oldWorker)
					}
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			node := obj.(*v1.Node)
			worker := Worker{
				MasterIP: cluster.MasterIP,
				HostIP:   node.Status.Addresses[0].Address,
			}
			log.Printf("delete event fetched for worker %v!\n", worker)
			index, found := findWorker(cluster.Workers, worker)
			if found {
				log.Printf("worker %v found in the cluster.Workers, removing...\n", worker)
				cluster.Workers = removeWorker(cluster.Workers, index)
				log.Printf("successfully removed worker %v from cluster.Workers slice!\n", worker)
				log.Printf("final cluster.Workers after delete operation = %v\n", cluster.Workers)
			} else {
				log.Printf("worker %v NOT found in the cluster.Workers, skipping remove operation!\n", worker)
			}
		},
	})
	informerFactory.Start(wait.NeverStop)
	informerFactory.WaitForCacheSync(wait.NeverStop)
}

// TODO: decrease cognitive complexity
func runServiceInformer(cluster *Cluster, clientSet *kubernetes.Clientset, customAnnotation, templateInputFile, templateOutputFile string) {
	informerFactory := informers.NewSharedInformerFactory(clientSet, time.Second * 30)
	serviceInformer := informerFactory.Core().V1().Services()
	serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			service := obj.(*v1.Service)
			_, ok := service.Annotations[customAnnotation]
			if service.Spec.Type == "NodePort" && ok {
				log.Printf("service %v is added on namespace %v with nodeport %v!\n", service.Name, service.Namespace,
					service.Spec.Ports[0].NodePort)

				nodePort := newNodePort(cluster.MasterIP, service.Spec.Ports[0].NodePort)
				index, found := findNodePort(cluster.NodePorts, *nodePort)
				if found {
					nodePort = cluster.NodePorts[index]
					for _, v := range cluster.Workers {
						index, found = findWorker(nodePort.Workers, *v)
						if !found {
							nodePort.Workers = append(nodePort.Workers, v)
						}
					}

					log.Printf("NodePort %v found in the backend.NodePorts, skipping adding...\n", nodePort)
				} else {
					for _, v := range cluster.Workers {
						index, found = findWorker(nodePort.Workers, *v)
						if !found {
							nodePort.Workers = append(nodePort.Workers, v)
						}
					}

					log.Printf("adding NodePort %v to the backend.NodePorts\n", nodePort)
					cluster.NodePorts = append(cluster.NodePorts, nodePort)
				}





				/*vserver := newVServer(nodePort)

				if !backend.isVServerExists(*vserver) {
					log.Printf("vserver %v not found in the backend.NodePorts, appending...\n", vserver)
					backend.VServers = append(backend.VServers, *vserver)
					log.Printf("final backend.VServers = %v\n", backend.VServers)
				} else {
					log.Printf("vserver %v already found in the backend.VServers, skipping...\n", vserver)
				}*/

				// Apply changes to the template
				tpl := template.Must(template.ParseFiles(templateInputFile))
				f, err := os.Create(templateOutputFile)
				checkError(err)


				err = tpl.ExecuteTemplate(f, "main", nginxConf)
				checkError(err)

				err = f.Close()
				checkError(err)

				//err = reloadNginx()
				//checkError(err)

				/*backend := newBackend(fmt.Sprintf("%s_%d", masterIp, nodePort), masterIp, nodePort)
					for i, v := range workerNodeIpAddr {
						worker := newWorker(int32(i), nodePort, v)
						log.Printf("appending worker %v:%v to backend %v\n", v, nodePort, backend.Name)
						backend.Workers = append(backend.Workers, *worker)
					}
					log.Printf("Adding backend %v to the &nginxConf.Backends\n", backend)
					addBackend(&nginxConf.Backends, *backend)

					vserver := newVServer(backend.Port, *backend)
					log.Printf("Adding vserver %v to the &nginxConf.VServers\n", vserver)
					addVserver(&nginxConf.VServers, *vserver)

					// Apply changes to the template
					tpl := template.Must(template.ParseFiles(templateInputFile))
					f, err := os.Create(templateOutputFile)
					checkError(err)

					err = tpl.Execute(f, &nginxConf)
					checkError(err)

					err = f.Close()
					checkError(err)

					//err = reloadNginx()
					//checkError(err)
				}*/
			}
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			/*oldService := oldObj.(*v1.Service)
			newService := newObj.(*v1.Service)
			// TODO: Handle the case that annotation is removed from the new service
			if oldService.Spec.Type == "NodePort" && oldService.ResourceVersion != newService.ResourceVersion {
				oldNodePort := oldService.Spec.Ports[0].NodePort
				newNodePort := newService.Spec.Ports[0].NodePort
				log.Printf("there is an update on the nodePort of the service %v on namespace %v!\n",
					oldService.Name, oldService.Namespace)
				oldBackend := newBackend(fmt.Sprintf("%s_%d", masterIp, oldNodePort), masterIp, oldNodePort)
				newBackend := newBackend(fmt.Sprintf("%s_%d", masterIp, newNodePort), masterIp, newNodePort)
				for i, v := range workerNodeIpAddr {
					oldBackend.Workers = append(oldBackend.Workers, *newWorker(int32(i), oldNodePort, v))
					newBackend.Workers = append(newBackend.Workers, *newWorker(int32(i), newNodePort, v))
				}

				oldVserver := newVServer(oldBackend.Port, *oldBackend)
				newVserver := newVServer(newBackend.Port, *newBackend)
				// Appending to the slices if annotation is found, removing if not found
				_, ok := newService.Annotations[customAnnotation]
				if ok {
					updateBackendsSlice(&nginxConf.Backends, *oldBackend, *newBackend)
					updateVServersSlice(&nginxConf.VServers, *oldVserver, *newVserver)
				} else {
					oldIndex, oldFound := findBackend(nginxConf.Backends, *oldBackend)
					if oldFound {
						nginxConf.Backends = removeFromBackendsSlice(nginxConf.Backends, oldIndex)
						nginxConf.VServers = removeFromVServersSlice(nginxConf.VServers, oldIndex)
					}
					newIndex, newFound := findBackend(nginxConf.Backends, *newBackend)
					if newFound {
						nginxConf.Backends = removeFromBackendsSlice(nginxConf.Backends, newIndex)
						nginxConf.VServers = removeFromVServersSlice(nginxConf.VServers, newIndex)
					}
				}
				// Apply changes to the template
				tpl := template.Must(template.ParseFiles(templateInputFile))
				f, err := os.Create(templateOutputFile)
				checkError(err)
				err = tpl.Execute(f, &nginxConf)
				checkError(err)
				err = f.Close()
				checkError(err)

				// err = reloadNginx()
				// checkError(err)
			}*/
		},
		DeleteFunc: func(obj interface{}) {
			/*
			service := obj.(*v1.Service)
			_, ok := service.Annotations[customAnnotation]
			if service.Spec.Type == "NodePort" && ok {
				nodePort := service.Spec.Ports[0].NodePort
				log.Printf("service %v is deleted on namespace %v!\n", service.Name, service.Namespace)

				// Create backend struct with nested K8sService
				backend := newBackend(fmt.Sprintf("%s_%d", masterIp, nodePort), masterIp, nodePort)
				for i, v := range workerNodeIpAddr {
					worker := newWorker(int32(i), nodePort, v)
					backend.Workers = append(backend.Workers, *worker)
				}
				index, found := findBackend(nginxConf.Backends, *backend)
				if found {
					nginxConf.Backends = removeFromBackendsSlice(nginxConf.Backends, index)
				}

				vserver := newVServer(nodePort, *backend)
				index, found = findVserver(nginxConf.VServers, *vserver)
				if found {
					nginxConf.VServers = removeFromVServersSlice(nginxConf.VServers, index)
				}
				// Apply changes to the template
				tpl := template.Must(template.ParseFiles(templateInputFile))
				f, err := os.Create(templateOutputFile)
				checkError(err)
				err = tpl.Execute(f, &nginxConf)
				checkError(err)
				err = f.Close()
				checkError(err)

				// err = reloadNginx()
				// checkError(err)
			}*/
		},
	})
	informerFactory.Start(wait.NeverStop)
	informerFactory.WaitForCacheSync(wait.NeverStop)
}