package main

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"log"
	"time"
	_ "time"
)

func runInformer(customAnnotation, templateInputFile, templateOutputFile, workerNodeIpAddr string, clientSet *kubernetes.Clientset) {
	var nodeportServices []K8sService
	// nginxConfPointer := &nginxConf

	informerFactory := informers.NewSharedInformerFactory(clientSet, time.Second * 30)
	serviceInformer := informerFactory.Core().V1().Services()
	serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if obj.(*v1.Service).Spec.Type == "NodePort" {
				service := K8sService{
					Namespace: obj.(*v1.Service).Namespace,
					Name: obj.(*v1.Service).Name,
					NodePort: obj.(*v1.Service).Spec.Ports[0].NodePort,
					Type: obj.(*v1.Service).Spec.Type,
				}
				_, found := findK8sService(nodeportServices, service)
				if !found {
					log.Printf("service %v not found in the nodeportServices slice, appending...\n", service)
					nodeportServices = append(nodeportServices, service)
				} else {
					log.Printf("service %v found in the nodeportServices slice, skipping appending...\n", service)
				}
			}
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			if oldObj.(*v1.Service).Spec.Type == "NodePort" && oldObj.(*v1.Service).ResourceVersion != newObj.(*v1.Service).ResourceVersion {
				oldService := K8sService{
					Namespace: oldObj.(*v1.Service).Namespace,
					Name: oldObj.(*v1.Service).Name,
					NodePort: oldObj.(*v1.Service).Spec.Ports[0].NodePort,
					Type: oldObj.(*v1.Service).Spec.Type,
				}

				newService := K8sService{
					Namespace: newObj.(*v1.Service).Namespace,
					Name: newObj.(*v1.Service).Name,
					NodePort: newObj.(*v1.Service).Spec.Ports[0].NodePort,
					Type: newObj.(*v1.Service).Spec.Type,
				}

				log.Printf("there is an update on the nodePort of the service %v on namespace %v!\nold service = %v\nnew service = %v\n",
					oldService.Name, oldService.Namespace, oldService, newService)

				nodeportServices = updateNodeportServices(nodeportServices, oldService, newService)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if obj.(*v1.Service).Spec.Type == "NodePort" {
				service := K8sService{
					Namespace: obj.(*v1.Service).Namespace,
					Name: obj.(*v1.Service).Name,
					NodePort: obj.(*v1.Service).Spec.Ports[0].NodePort,
					Type: obj.(*v1.Service).Spec.Type,
				}

				i, found := findK8sService(nodeportServices, service)
				if found {
					log.Printf("service %v found in the nodeportServices slice, deleting it from slice...\n", service)
					nodeportServices = removeFromNodeportServices(nodeportServices, i)
					log.Printf("targetServices slice after deletion = %v\n", nodeportServices)
				} else {
					log.Printf("service %v not found in the nodeportServices slice, skipping deletion...\n", service)
				}
			}
		},
	})
	informerFactory.Start(wait.NeverStop)
	informerFactory.WaitForCacheSync(wait.NeverStop)

	/*targetNamespaces, err := getNamespaces(clientSet)
	if err != nil {
		log.Printf("an error occured while fetching the namespaces from the kube-apiserver, aborting scheduled execution!")
	} else {
		for _, ns := range targetNamespaces {
			services, err := getServices(ns.Name, clientSet)
			if err != nil {
				log.Printf("an error occured while fetching the services from the kube-apiserver, skipping namespace %s\n", ns.Name)
			} else {
				for _, svc := range services {
					if svc.Spec.Type == "NodePort" && svc.Annotations[customAnnotation] == "true" {
						targetPorts = append(targetPorts, svc.Spec.Ports[0].NodePort)
						backend := Backend{
							Name: fmt.Sprintf("%s_%d", workerNodeIpAddr, svc.Spec.Ports[0].NodePort),
							IP: workerNodeIpAddr,
							Port: svc.Spec.Ports[0].NodePort,
						}
						_, found := findBackend(nginxConfPointer.Backends, backend)
						if !found {
							nginxConfPointer.Backends = append(nginxConfPointer.Backends, backend)
						}

						vserver := VServer{
							Port:    svc.Spec.Ports[0].NodePort,
							Backend: backend,
						}
						_, found = findVserver(nginxConfPointer.VServers, vserver)
						if !found {
							nginxConfPointer.VServers = append(nginxConfPointer.VServers, vserver)
						}
					}
				}
			}
		}

		tpl := template.Must(template.ParseFiles(templateInputFile))
		f, err := os.Create(templateOutputFile)
		checkError(err)

		sync(targetPorts, workerNodeIpAddr)
		err = tpl.Execute(f, &nginxConfPointer)
		checkError(err)

		err = f.Close()
		checkError(err)

		err = reloadNginx()
		checkError(err)
	}*/
}

// Sync nginxConfPointer.VServers with targetPorts
// TODO: Fix possible concurrency problem here
func sync(targetPorts []int32, workerNodeIpAddr string) {
	var tmpVservers []VServer
	var tmpBackends []Backend
	nginxConfPointer := &nginxConf
	for _, v := range targetPorts {
		backend := Backend{
			Name: fmt.Sprintf("%s_%d", workerNodeIpAddr, v),
			IP: workerNodeIpAddr,
			Port: v,
		}

		vserver := VServer{
			Port:    v,
			Backend: backend,
		}

		tmpBackends = append(tmpBackends, backend)
		tmpVservers = append(tmpVservers, vserver)
	}

	for i, v := range nginxConfPointer.Backends {
		if v.IP == workerNodeIpAddr {
			_, found := findBackend(tmpBackends, v)
			if !found {
				nginxConfPointer.Backends[i] = nginxConfPointer.Backends[len(nginxConfPointer.Backends) - 1]
				nginxConfPointer.Backends = nginxConfPointer.Backends[:len(nginxConfPointer.Backends) - 1]
			}
		}
	}

	for i, v := range nginxConfPointer.VServers {
		if v.Backend.IP == workerNodeIpAddr {
			_, found := findVserver(tmpVservers, v)
			if !found {
				nginxConfPointer.VServers[i] = nginxConfPointer.VServers[len(nginxConfPointer.VServers) - 1]
				nginxConfPointer.VServers = nginxConfPointer.VServers[:len(nginxConfPointer.VServers) - 1]
			}
		}
	}
}

func getNamespaces(clientSet *kubernetes.Clientset) ([]v1.Namespace, error) {
	namespaces, err := clientSet.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return namespaces.Items, nil
}

func getServices(namespace string, clientSet *kubernetes.Clientset) ([]v1.Service, error) {
	services, err := clientSet.CoreV1().Services(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return services.Items, nil
}