package main

import (
	"flag"
	"k8s.io/client-go/kubernetes"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	clientSet *kubernetes.Clientset
)

func main() {
	kubeConfigPath := flag.String("kubeConfigPath", filepath.Join(os.Getenv("HOME"), ".kube", "config"),
		"comma seperated list of kubeconfig file paths to access with the cluster")
	tickerIntervalSeconds := flag.Int("tickerIntervalSeconds", 10, "how frequently scheduled job will " +
		"run on kubernetes cluster to fetch services")
	customAnnotation := flag.String("customAnnotation", "hayde.trendyol.io/enabled", "annotation to specify " +
		"selectable services")
	workerNodeIpAddresses := flag.String("workerNodeIpAddresses", "192.168.0.201", "comma seperated ip address of the worker nodes " +
		"to reach the services over NodePort")
	flag.Parse()

	// initialize kube client
	restConfig, err := getConfig(*kubeConfigPath)
	checkError(err)
	clientSet, err = getClientSet(restConfig)
	checkError(err)

	workerNodeIPs := strings.Split(*workerNodeIpAddresses, ",")
	for _, v := range workerNodeIPs {
		backend := Backend{
			Name: v,
			IP: v,
		}

		go func(customAnnotation string, backend Backend) {
			runScheduledJob(customAnnotation, v, backend)
			ticker := time.NewTicker(time.Duration(int32(*tickerIntervalSeconds)) * time.Second)
			for _ = range ticker.C {
				runScheduledJob(customAnnotation, v, backend)
			}
		}(*customAnnotation, backend)
	}

	select {}
}