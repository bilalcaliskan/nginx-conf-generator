package main

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"os/exec"
)

func getConfig(kubeConfigPath string) (*rest.Config, error) {
	var config *rest.Config
	var err error

	config, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func getClientSet(config *rest.Config) (*kubernetes.Clientset, error) {
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientSet, nil
}

func checkError(err error) {
	if err != nil {
		log.Fatalln(err.Error())
	}
}

// findItem takes a slice and looks for an element in it. If found it will
// return it's key, otherwise it will return -1 and a bool of false.
func findVserver(vservers []VServer, vserver VServer) (int, bool) {
	for i, item := range vservers {
		if item == vserver {
			return i, true
		}
	}
	return -1, false
}

func findK8sService(targetServices []K8sService, service K8sService) (int, bool) {
	for i, item := range targetServices {
		if item == service {
			return i, true
		}
	}
	return -1, false
}

func findBackend(backends []Backend, backend Backend) (int, bool) {
	for i, item := range backends {
		if item == backend {
			return i, true
		}
	}
	return -1, false
}

func reloadNginx() error {
	cmd := exec.Command("nginx", "-s", "reload")
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func indexOf(element interface{}, data []interface{}) int {
	for k, v := range data {
		if element == v {
			return k
		}
	}
	return -1
}

func removeFromNodeportServices(slice []K8sService, index int) []K8sService {
	return append(slice[:index], slice[index+1:]...)
}

func updateNodeportServices(slice []K8sService, oldService K8sService, newService K8sService) []K8sService {
	oldIndex, oldFound := findK8sService(slice, oldService)
	if oldFound {
		log.Printf("update operation is starting on the nodeportServices slice %v\n", slice)
		log.Printf("removing service %v from nodeportServices slice!\n", oldService)
		slice = removeFromNodeportServices(slice, oldIndex)
		_, newFound := findK8sService(slice, newService)
		if !newFound {
			log.Printf("adding service %v to targetServices slice!\n", newService)
			slice = append(slice, newService)
		} else {
			log.Printf("new service %v already found in the targetServices slice, skipping insertion...\n", newService)
		}
	} else {
		log.Printf("old service %v not found in the targetServices slice, skipping insertion, instead adding the new one %v...\n",
			oldService, newService)
		slice = append(slice, newService)
	}
	log.Printf("final nodeportServices slice after update operation = %v\n", slice)
	return slice
}