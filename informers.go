package main

import (
	"fmt"
	"html/template"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"log"
	"os"
	"time"
	_ "time"
)

// TODO: Implement that informer to update a pointer on workerNodeIpAddr []string about node changes(added, removed etc)
func runNodeInformer(masterIp string, clientSet *kubernetes.Clientset) {
	informerFactory := informers.NewSharedInformerFactory(clientSet, time.Second * 30)
	nodeInformer := informerFactory.Core().V1().Nodes()
	nodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    nil,
		UpdateFunc: nil,
		DeleteFunc: nil,
	})
}

func runServiceInformer(customAnnotation, templateInputFile, templateOutputFile, masterIp string, workerNodeIpAddr []string, clientSet *kubernetes.Clientset) {
	informerFactory := informers.NewSharedInformerFactory(clientSet, time.Second * 30)
	serviceInformer := informerFactory.Core().V1().Services()
	serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			service := obj.(*v1.Service)
			_, ok := service.Annotations[customAnnotation]
			if service.Spec.Type == "NodePort" && ok {
				nodePort := service.Spec.Ports[0].NodePort
				/*log.Printf("service %v is added on namespace %v with nodeport %v!\n", service.Name, service.Namespace,
					nodePort)*/

				backend := newBackend(fmt.Sprintf("%s_%d", masterIp, nodePort), masterIp, nodePort)
				for i, v := range workerNodeIpAddr {
					worker := newWorker(int32(i), nodePort, v)
					log.Printf("appending worker %v:%v to backend %v\n", v, nodePort, backend.Name)
					backend.Workers = append(backend.Workers, *worker)
				}
				addBackend(&nginxConf.Backends, *backend)

				vserver := newVServer(backend.Port, *backend)
				addVserver(&nginxConf.VServers, *vserver)
				log.Printf("Workers of backend %v = backend.Workers = %v\n", backend.Name, backend.Workers)


				log.Printf("final nginxConf.VServers = %v\nfinal nginxConf.Backends = %v\n", nginxConf.VServers,
					nginxConf.Backends)

				// Apply changes to the template
				tpl := template.Must(template.ParseFiles(templateInputFile))
				f, err := os.Create(templateOutputFile)
				checkError(err)

				err = tpl.Execute(f, &nginxConf)
				checkError(err)

				err = f.Close()
				checkError(err)

				/*err = reloadNginx()
				checkError(err)*/
			}
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			oldService := oldObj.(*v1.Service)
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
					nginxConf.Backends = updateBackendsSlice(nginxConf.Backends, *oldBackend, *newBackend)
					nginxConf.VServers = updateVserversSlice(nginxConf.VServers, *oldVserver, *newVserver)
				} else {
					oldIndex, oldFound := findBackend(nginxConf.Backends, *oldBackend)
					if oldFound {
						nginxConf.Backends = removeFromBackendsSlice(nginxConf.Backends, oldIndex)
						nginxConf.VServers = removeFromVserversSlice(nginxConf.VServers, oldIndex)
					}
					newIndex, newFound := findBackend(nginxConf.Backends, *newBackend)
					if newFound {
						nginxConf.Backends = removeFromBackendsSlice(nginxConf.Backends, newIndex)
						nginxConf.VServers = removeFromVserversSlice(nginxConf.VServers, newIndex)
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

				/*err = reloadNginx()
				checkError(err)*/
			}
		},
		DeleteFunc: func(obj interface{}) {
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
					nginxConf.VServers = removeFromVserversSlice(nginxConf.VServers, index)
				}
				// Apply changes to the template
				tpl := template.Must(template.ParseFiles(templateInputFile))
				f, err := os.Create(templateOutputFile)
				checkError(err)
				err = tpl.Execute(f, &nginxConf)
				checkError(err)
				err = f.Close()
				checkError(err)

				/*err = reloadNginx()
				checkError(err)*/
			}
		},
	})
	informerFactory.Start(wait.NeverStop)
	informerFactory.WaitForCacheSync(wait.NeverStop)
}