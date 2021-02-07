package main

import (
	"flag"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"strings"
)

var (
	clusters []*Cluster
	nginxConf = &NginxConf{
		Clusters: clusters,
	}
	kubeConfigPaths, templateInputFile, templateOutputFile, customAnnotation, workerNodeLabel *string
	logger *zap.Logger
	err error
	kubeConfigPathArr []string
)

func init() {
	logger, err = zap.NewProduction()
	if err != nil {
		panic(err)
	}

	kubeConfigPaths = flag.String("kubeConfigPaths", filepath.Join(os.Getenv("HOME"), ".kube", "minikubeconfig"),
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

	kubeConfigPathArr = strings.Split(*kubeConfigPaths, ",")
}

func main() {
	// TODO: Unit testing!

	for _, path := range kubeConfigPathArr {
		restConfig, err := getConfig(path)
		if err != nil {
			logger.Fatal("fatal error occured while getting k8s config", zap.String("error", err.Error()))
		}

		clientSet, err := getClientSet(restConfig)
		if err != nil {
			logger.Fatal("fatal error occured while getting clientset", zap.String("error", err.Error()))
		}

		masterIp := strings.Split(strings.Split(restConfig.Host, "//")[1], ":")[0]
		cluster := newCluster(masterIp, make([]*Worker, 0))
		nginxConf.Clusters = append(nginxConf.Clusters, cluster)

		runNodeInformer(cluster, clientSet, logger)
		runServiceInformer(cluster, clientSet, logger)
	}

	select {}
}