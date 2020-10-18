package main

import (
	"html/template"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"os"
	_ "time"
)

func runScheduledJob(customAnnotation, nodePortIp string, backend Backend) {
	var targetPorts []int32
	/*log.Println("starting scheduling execution to fetch NodePort type services on all namespaces from kube-apiserver...")
	beforeExecution := time.Now()*/
	targetNamespaces, err := getNamespaces()
	if err != nil {
		log.Printf("an error occured while fetching the namespaces from the kube-apiserver, aborting scheduled execution!")
	} else {
		for _, ns := range targetNamespaces {
			services, err := getServices(ns.Name)
			if err != nil {
				log.Printf("an error occured while fetching the services from the kube-apiserver, skipping namespace %s\n", ns.Name)
			} else {
				for _, svc := range services {
					if svc.Spec.Type == "NodePort" && svc.Annotations[customAnnotation] == "true" {
						log.Printf("Service %s on namespace %s has required annotation and required service type!\n",
							svc.Name, ns.Name)
						targetPorts = append(targetPorts, svc.Spec.Ports[0].NodePort)
					}
				}
			}
		}
		/*log.Printf("ending scheduling execution. Fetched %d NodePort type services on %d namespaces from kube-apiserver, " +
			"took %d ms!\n", len(targetPorts), len(targetNamespaces), time.Now().Sub(beforeExecution).Milliseconds())*/
		
		var vservers []VServer
		for _, v := range targetPorts {
			vserver := VServer{
				Port:    v,
				Backend: backend,
			}
			vservers = append(vservers, vserver)
		}

		nginxConf := NginxConf{
			VServers: vservers,
			Backend:  backend,
		}

		tpl := template.Must(template.ParseFiles("resources/default.conf"))

		f, err := os.Create("./myfile")
		checkError(err)

		err = tpl.Execute(f, &nginxConf)
		checkError(err)

		err = f.Close()
		checkError(err)
	}
}

func getNamespaces() ([]v1.Namespace, error) {
	namespaces, err := clientSet.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return namespaces.Items, nil
}

func getServices(namespace string) ([]v1.Service, error) {
	services, err := clientSet.CoreV1().Services(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return services.Items, nil
}