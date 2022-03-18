package main

import (
	"github.com/dimiro1/banner"
	"go.uber.org/zap"
	"io/ioutil"
	"nginx-conf-generator/internal/k8s/informers"
	"nginx-conf-generator/internal/k8s/types"
	"nginx-conf-generator/internal/logging"
	"nginx-conf-generator/internal/metrics"
	"nginx-conf-generator/internal/options"
	"os"
	"strings"
)

var (
	clusters          []*types.Cluster
	nginxConf         = types.NewNginxConf(clusters)
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
		restConfig, err := informers.GetConfig(path)
		if err != nil {
			logger.Fatal("fatal error occurred while getting k8s config", zap.String("error", err.Error()))
		}

		clientSet, err := informers.GetClientSet(restConfig)
		if err != nil {
			logger.Fatal("fatal error occurred while getting clientset", zap.String("error", err.Error()))
		}

		masterIp := strings.Split(strings.Split(restConfig.Host, "//")[1], ":")[0]
		cluster := types.NewCluster(masterIp, make([]*types.Worker, 0))
		nginxConf.Clusters = append(nginxConf.Clusters, cluster)
		logger := logging.NewLogger()
		logger.With(zap.String("masterIP", cluster.MasterIP))

		informers.RunNodeInformer(cluster, clientSet, logger, ncgo, nginxConf)
		informers.RunServiceInformer(cluster, clientSet, logger, ncgo, nginxConf)
	}

	go func() {
		if err := metrics.RunMetricsServer(); err != nil {
			logger.Fatal("fatal error occurred while spinning up metrics server",
				zap.String("error", err.Error()))
		}
	}()

	select {}
}
