package main

import (
	"flag"
	// v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"log"
	"os"
	"path/filepath"
	"strings"
)

var nginxConf NginxConf


func main() {
	kubeConfigPaths := flag.String("kubeConfigPaths", filepath.Join(os.Getenv("HOME"), ".kube", "minikubeconfig"),
		"comma seperated list of kubeconfig file paths to access with the cluster")
	workerNodeLabel := flag.String("workerNodeLabel", "node-role.kubernetes.io/worker", "label to specify " +
		"worker nodes, defaults to node-role.kubernetes.io/worker=")
	// customAnnotation := flag.String("customAnnotation", "nginx-conf-generator/enabled", "annotation to specify " +
	//	"selectable services")
	//templateInputFile := flag.String("templateInputFile", "resources/default.conf.tmpl", "input " +
	//	"path of the template file")
	//templateOutputFile := flag.String("templateOutputFile", "/etc/nginx/sites-enabled/default", "output " +
	//	"path of the template file")
	// templateOutputFile := flag.String("templateOutputFile", "default", "output path of the template file")
	flag.Parse()

	// TODO: create shared informer for nodes, handle the case that a worker is removed or any worker added to the cluster
	// TODO: fix the performance-related problems, use more pointers to avoid re-initializing slices etc

	kubeConfigPathArr := strings.Split(*kubeConfigPaths, ",")
	for _, cluster := range kubeConfigPathArr {
		restConfig, err := getConfig(cluster)
		checkError(err)
		clientSet, err := getClientSet(restConfig)
		checkError(err)
		/*workerNodeList, err := clientSet.CoreV1().Nodes().List(v1.ListOptions{
			LabelSelector: *workerNodeLabel,
		})*/
		checkError(err)

		/*var workerNodeIps []string
		for _, v := range workerNodeList.Items {
			workerNodeIps = append(workerNodeIps, v.Status.Addresses[0].Address)
		}*/

		masterIp := strings.Split(strings.Split(restConfig.Host, "//")[1], ":")[0]
		//log.Printf("workerNodeIps on cluster %v = %v\n", masterIp, workerNodeIps)

		backend := Backend{
			MasterIP:  masterIp,
			Workers:   make([]Worker, 0),
			NodePorts: make([]int32, 0),
		}
		runNodeInformer(&backend, clientSet, *workerNodeLabel)
		// runServiceInformer(*customAnnotation, *templateInputFile, *templateOutputFile, masterIp, workerNodeIps, clientSet)
	}

	select {}
}