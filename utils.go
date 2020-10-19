package main

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"log"
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
func findItem(vservers []VServer, vserver VServer) (int, bool) {
	for i, item := range vservers {
		if item == vserver {
			return i, true
		}
	}
	return -1, false
}