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

func removeFromBackendsSlice(slice []Backend, index int) []Backend {
	return append(slice[:index], slice[index+1:]...)
}

func updateBackendsSlice(slice []Backend, oldBackend Backend, newBackend Backend) []Backend {
	oldIndex, oldFound := findBackend(slice, oldBackend)
	if oldFound {
		log.Printf("update operation is starting on the nginxConfPointer.Backends slice %v\n", slice)
		log.Printf("removing backend %v from nginxConfPointer.Backends slice!\n", oldBackend)
		slice = removeFromBackendsSlice(slice, oldIndex)

		_, newFound := findBackend(slice, newBackend)
		if !newFound {
			log.Printf("adding backend %v to nginxConfPointer.Backends slice!\n", newBackend)
			slice = append(slice, newBackend)
		} else {
			log.Printf("new backend %v already found in the nginxConfPointer.Backends slice, skipping insertion...\n", newBackend)
		}
	} else {
		log.Printf("old backend %v not found in the nginxConfPointer.Backends slice, skipping insertion, instead adding the new one %v...\n",
			oldBackend, newBackend)
		slice = append(slice, newBackend)
	}
	log.Printf("final nginxConfPointer.Backends slice after update operation = %v\n", slice)
	return slice
}

func removeFromVserversSlice(slice []VServer, index int) []VServer {
	return append(slice[:index], slice[index+1:]...)
}

func updateVserversSlice(slice []VServer, oldVserver VServer, newVserver VServer) []VServer {
	oldIndex, oldFound := findVserver(slice, oldVserver)
	if oldFound {
		log.Printf("update operation is starting on the nginxConfPointer.Vservers slice %v\n", slice)
		log.Printf("removing vserver %v from nginxConfPointer.Vservers slice!\n", oldVserver)
		slice = removeFromVserversSlice(slice, oldIndex)

		_, newFound := findVserver(slice, newVserver)
		if !newFound {
			log.Printf("adding vserver %v to nginxConfPointer.Vservers slice!\n", newVserver)
			slice = append(slice, newVserver)
		} else {
			log.Printf("new vserver %v already found in the nginxConfPointer.Vservers slice, skipping insertion...\n", newVserver)
		}
	} else {
		log.Printf("old vserver %v not found in the nginxConfPointer.Vservers slice, skipping insertion, instead adding the new one %v...\n",
			oldVserver, newVserver)
		slice = append(slice, newVserver)
	}
	log.Printf("final nginxConfPointer.Vservers slice after update operation = %v\n", slice)
	return slice
}