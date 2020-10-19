package main

import (
	"html/template"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"log"
	"os"
	_ "time"
)

func runScheduledJob(customAnnotation, templateOutputFile string, backend Backend, clientSet *kubernetes.Clientset) {
	var targetPorts []int32
	nginxConfPointer := &nginxConf
	targetNamespaces, err := getNamespaces(clientSet)
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
					}
				}
			}
		}
		/*log.Printf("ending scheduling execution. Fetched %d NodePort type services on %d namespaces from kube-apiserver, " +
			"took %d ms!\n", len(targetPorts), len(targetNamespaces), time.Now().Sub(beforeExecution).Milliseconds())*/

		for _, v := range targetPorts {
			vserver := VServer{
				Port:    v,
				Backend: backend,
			}

			_, found := findItem(nginxConfPointer.VServers, vserver)
			if !found {
				nginxConfPointer.VServers = append(nginxConfPointer.VServers, vserver)
			}
		}



		tpl := template.Must(template.ParseFiles("resources/default.conf"))
		f, err := os.Create(templateOutputFile)
		checkError(err)

		sync(targetPorts, backend)
		err = tpl.Execute(f, &nginxConfPointer)
		checkError(err)

		err = f.Close()
		checkError(err)

		err = reloadNginx()
		checkError(err)
	}
}

// Sync nginxConfPointer.VServers with targetPorts
// TODO: Fix possible concurrency problem here
func sync(targetPorts []int32, backend Backend) {
	var tmpVservers []VServer
	nginxConfPointer := &nginxConf
	for _, v := range targetPorts {
		vserver := VServer{
			Port:    v,
			Backend: backend,
		}
		tmpVservers = append(tmpVservers, vserver)
	}

	for i, v := range nginxConfPointer.VServers {
		if v.Backend == backend {
			_, found := findItem(tmpVservers, v)
			if !found {
				// service label or type changed, remove from nginxConfPointer.VServers
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