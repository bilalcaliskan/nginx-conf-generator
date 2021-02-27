package main

import (
	flag "github.com/spf13/pflag"
	"go.uber.org/zap"
	"nginx-conf-generator/pkg/kubernetes"
	"os"
	"path/filepath"
	"strings"
)

var (
	clusters []*kubernetes.Cluster
	nginxConf = &kubernetes.NginxConf{
		Clusters: clusters,
	}
	kubeConfigPaths, templateInputFile, templateOutputFile, customAnnotation, workerNodeLabel string
	logger *zap.Logger
	err error
	kubeConfigPathArr []string
)

func init() {
	logger, err = zap.NewProduction()
	if err != nil {
		panic(err)
	}

	flag.StringVar(&kubeConfigPaths, "kubeConfigPaths", filepath.Join(os.Getenv("HOME"), ".kube", "minikubeconfig"),
		"comma seperated list of kubeconfig file paths to access with the cluster")
	flag.StringVar(&workerNodeLabel, "workerNodeLabel", "node-role.kubernetes.io/worker", "label to specify " +
		"worker nodes, defaults to node-role.kubernetes.io/worker=")
	flag.StringVar(&customAnnotation, "customAnnotation", "nginx-conf-generator/enabled", "annotation to specify " +
		"selectable services")
	flag.StringVar(&templateInputFile, "templateInputFile", "resources/default.conf.tmpl", "input " +
		"path of the template file")
	flag.StringVar(&templateOutputFile, "templateOutputFile", "/etc/nginx/sites-enabled/default", "output " +
		"path of the template file")
	flag.Parse()

	kubeConfigPathArr = strings.Split(kubeConfigPaths, ",")
}

func main() {
	// TODO: Unit testing!

	for _, path := range kubeConfigPathArr {
		restConfig, err := kubernetes.GetConfig(path)
		if err != nil {
			logger.Fatal("fatal error occured while getting k8s config", zap.String("error", err.Error()))
		}

		clientSet, err := kubernetes.GetClientSet(restConfig)
		if err != nil {
			logger.Fatal("fatal error occured while getting clientset", zap.String("error", err.Error()))
		}

		masterIp := strings.Split(strings.Split(restConfig.Host, "//")[1], ":")[0]
		cluster := kubernetes.NewCluster(masterIp, make([]*kubernetes.Worker, 0))
		nginxConf.Clusters = append(nginxConf.Clusters, cluster)

		kubernetes.RunNodeInformer(cluster, clientSet, logger, workerNodeLabel, templateInputFile, templateOutputFile,
			nginxConf)
		kubernetes.RunServiceInformer(cluster, clientSet, logger, customAnnotation, templateInputFile, templateOutputFile,
			nginxConf)
	}

	select {}
}