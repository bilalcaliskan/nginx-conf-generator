package main

import (
	"flag"
	"k8s.io/client-go/kubernetes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var nginxConf NginxConf

func main() {
	kubeConfigPaths := flag.String("kubeConfigPaths", filepath.Join(os.Getenv("HOME"), ".kube", "config"),
		"comma seperated list of kubeconfig file paths to access with the cluster")
	tickerIntervalSeconds := flag.Int("tickerIntervalSeconds", 10, "how frequently scheduled job will " +
		"run on kubernetes cluster to fetch services")
	customAnnotation := flag.String("customAnnotation", "hayde.trendyol.io/enabled", "annotation to specify " +
		"selectable services")
	// single worker node ip address for each cluster. order of ip addresses must be same with kubeConfigPaths[]
	workerNodeIps := flag.String("workerNodeIps", "192.168.0.201", "comma seperated ip " +
		"address of the worker nodes to reach the services over NodePort")
	templateInputFile := flag.String("templateInputFile", "/opt/resources/default.conf.tmpl", "input " +
		"path of the template file")
	templateOutputFile := flag.String("templateOutputFile", "/etc/nginx/sites-enabled/default", "output " +
		"path of the template file")
	flag.Parse()

	kubeConfigPathArr := strings.Split(*kubeConfigPaths, ",")
	workerNodeIpArr := strings.Split(*workerNodeIps, ",")
	for i, v := range workerNodeIpArr {
		// initialize kube client
		restConfig, err := getConfig(kubeConfigPathArr[i])
		checkError(err)
		clientSet, err := getClientSet(restConfig)
		checkError(err)

		go func(customAnnotation, templateInputFile, templateOutputFile, workerNodeIpAddr string, clientSet *kubernetes.Clientset) {
			runScheduledJob(customAnnotation, templateInputFile, templateOutputFile, workerNodeIpAddr, clientSet)
			ticker := time.NewTicker(time.Duration(int32(*tickerIntervalSeconds)) * time.Second)
			for range ticker.C {
				runScheduledJob(customAnnotation, templateInputFile, templateOutputFile, workerNodeIpAddr, clientSet)
			}
		}(*customAnnotation, *templateInputFile, *templateOutputFile, v, clientSet)
	}

	log.Printf("nginxConf.Backends = %v\n", nginxConf.Backends)
	select {}
}