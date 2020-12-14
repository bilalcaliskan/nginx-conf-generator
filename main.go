package main

import (
	"flag"
	// v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"log"
	"os"
	"path/filepath"
	"strings"
)

var (
	clusters []*Cluster
	nginxConf = &NginxConf{
		Clusters: clusters,
	}
	templateInputFile, templateOutputFile, customAnnotation, workerNodeLabel *string
)


func main() {
	kubeConfigPaths := flag.String("kubeConfigPaths", filepath.Join(os.Getenv("HOME"), ".kube", "minikubeconfig"),
		"comma seperated list of kubeconfig file paths to access with the cluster")
	workerNodeLabel = flag.String("workerNodeLabel", "node-role.kubernetes.io/worker", "label to specify " +
		"worker nodes, defaults to node-role.kubernetes.io/worker=")
	customAnnotation = flag.String("customAnnotation", "nginx-conf-generator/enabled", "annotation to specify " +
		"selectable services")
	templateInputFile = flag.String("templateInputFile", "resources/default.conf.tmpl", "input " +
		"path of the template file")
	// templateOutputFile = flag.String("templateOutputFile", "/etc/nginx/sites-enabled/default", "output " +
	//	"path of the template file")
	templateOutputFile = flag.String("templateOutputFile", "default", "output path of the template file")
	flag.Parse()

	// TODO: Fix the performance-related problems, use more pointers to avoid re-initializing slices etc
	// TODO: Refactor neccessary parts
	// TODO: Test multi-cluster
	// TODO: Unit testing!

	kubeConfigPathArr := strings.Split(*kubeConfigPaths, ",")
	for _, path := range kubeConfigPathArr {
		restConfig, err := getConfig(path)
		checkError(err)
		clientSet, err := getClientSet(restConfig)
		checkError(err)

		masterIp := strings.Split(strings.Split(restConfig.Host, "//")[1], ":")[0]
		cluster := newCluster(masterIp, make([]*Worker, 0))
		nginxConf.Clusters = append(nginxConf.Clusters, cluster)

		// run nodeInformer with seperate goroutine
		runNodeInformer(cluster, clientSet)

		// run serviceInformer with seperate goroutine
		runServiceInformer(cluster, clientSet)
	}

	select {}
}