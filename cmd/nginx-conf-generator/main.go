package main

import (
	"github.com/dimiro1/banner"
	"go.uber.org/zap"
	"io/ioutil"
	"nginx-conf-generator/internal/k8s"
	"nginx-conf-generator/internal/logging"
	"nginx-conf-generator/internal/metrics"
	"nginx-conf-generator/internal/options"
	"os"
	"strings"
)

var (
	clusters  []*k8s.Cluster
	nginxConf = &k8s.NginxConf{
		Clusters: clusters,
	}
	ncgo              *options.NginxConfGeneratorOptions
	logger            *zap.Logger
	kubeConfigPathArr []string
)

func init() {
	logger = logging.GetLogger()
	ncgo = options.GetNginxConfGeneratorOptions()
	kubeConfigPathArr = strings.Split(ncgo.KubeConfigPaths, ",")

	bannerBytes, _ := ioutil.ReadFile("banner.txt")
	banner.Init(os.Stdout, true, false, strings.NewReader(string(bannerBytes)))
}

func main() {
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

		k8s.RunNodeInformer(cluster, clientSet, ncgo, nginxConf)
		k8s.RunServiceInformer(cluster, clientSet, ncgo, nginxConf)
	}

	go func() {
		if err := metrics.RunMetricsServer(); err != nil {
			logger.Fatal("fatal error occurred while spinning up metrics server",
				zap.String("error", err.Error()))
		}
	}()

	select {}
}
