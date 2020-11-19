package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var nginxConf NginxConf

func main() {
	kubeConfigPaths := flag.String("kubeConfigPaths", filepath.Join(os.Getenv("HOME"), ".kube", "minikubeconfig"),
		"comma seperated list of kubeconfig file paths to access with the cluster")
	customAnnotation := flag.String("customAnnotation", "nginx-conf-generator/enabled", "annotation to specify " +
		"selectable services")
	// single worker node ip address for each cluster. order of ip addresses must be same with kubeConfigPaths[]
	workerNodeIps := flag.String("workerNodeIps", "192.168.99.101", "comma seperated ip " +
		"address of the worker nodes to reach the services over NodePort")
	templateInputFile := flag.String("templateInputFile", "/opt/resources/default.conf.tmpl", "input " +
		"path of the template file")
	templateOutputFile := flag.String("templateOutputFile", "/etc/nginx/sites-enabled/default", "output " +
		"path of the template file")
	flag.Parse()

	kubeConfigPathArr := strings.Split(*kubeConfigPaths, ",")
	workerNodeIpArr := strings.Split(*workerNodeIps, ",")
	for index, ip := range workerNodeIpArr {
		// initialize kube client
		restConfig, err := getConfig(kubeConfigPathArr[index])
		checkError(err)
		clientSet, err := getClientSet(restConfig)
		checkError(err)
		runInformer(*customAnnotation, *templateInputFile, *templateOutputFile, ip, clientSet)
	}

	log.Printf("nginxConf.Backends = %v\n", nginxConf.Backends)
	select {}
}