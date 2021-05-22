package main

import (
	flag "github.com/spf13/pflag"
	"go.uber.org/zap"
	"nginx-conf-generator/pkg/k8s"
	"os"
	"path/filepath"
	"strings"
)

var (
	clusters  []*k8s.Cluster
	nginxConf = &k8s.NginxConf{
		Clusters: clusters,
	}
	kubeConfigPaths, templateInputFile, templateOutputFile, customAnnotation, workerNodeLabel string
	logger                                                                                    *zap.Logger
	err                                                                                       error
	kubeConfigPathArr                                                                         []string
)

func init() {
	logger, err = zap.NewProduction()
	if err != nil {
		panic(err)
	}

	flag.StringVar(&kubeConfigPaths, "kubeConfigPaths", filepath.Join(os.Getenv("HOME"), ".kube", "minikubeconfig"),
		"comma separated list of kubeconfig file paths to access with the cluster")
	flag.StringVar(&workerNodeLabel, "workerNodeLabel", "node-role.k8s.io/worker", "label to specify "+
		"worker nodes, defaults to node-role.k8s.io/worker=")
	flag.StringVar(&customAnnotation, "customAnnotation", "nginx-conf-generator/enabled", "annotation to specify "+
		"selectable services")
	flag.StringVar(&templateInputFile, "templateInputFile", "resources/default.conf.tmpl", "input "+
		"path of the template file")
	flag.StringVar(&templateOutputFile, "templateOutputFile", "/etc/nginx/sites-enabled/default", "output "+
		"path of the template file")
	flag.Parse()

	kubeConfigPathArr = strings.Split(kubeConfigPaths, ",")
}

func main() {
	// TODO: Unit testing!

	defer func() {
		err := logger.Sync()
		if err != nil {
			panic(err)
		}
	}()

	for _, path := range kubeConfigPathArr {
		restConfig, err := k8s.GetConfig(path)
		if err != nil {
			logger.Fatal("fatal error occurred while getting k8s config", zap.String("error", err.Error()))
		}

		clientSet, err := k8s.GetClientSet(restConfig)
		if err != nil {
			logger.Fatal("fatal error occurred while getting clientset", zap.String("error", err.Error()))
		}

		masterIp := strings.Split(strings.Split(restConfig.Host, "//")[1], ":")[0]
		cluster := k8s.NewCluster(masterIp, make([]*k8s.Worker, 0))
		nginxConf.Clusters = append(nginxConf.Clusters, cluster)

		k8s.RunNodeInformer(cluster, clientSet, logger, workerNodeLabel, templateInputFile, templateOutputFile,
			nginxConf)
		k8s.RunServiceInformer(cluster, clientSet, logger, customAnnotation, templateInputFile, templateOutputFile,
			nginxConf)
	}

	select {}
}
