package cmd

import (
	"github.com/bilalcaliskan/nginx-conf-generator/internal/k8s/informers"
	"github.com/bilalcaliskan/nginx-conf-generator/internal/k8s/types"
	"github.com/bilalcaliskan/nginx-conf-generator/internal/logging"
	"github.com/bilalcaliskan/nginx-conf-generator/internal/metrics"
	"github.com/bilalcaliskan/nginx-conf-generator/internal/options"
	"github.com/bilalcaliskan/nginx-conf-generator/internal/version"
	"github.com/dimiro1/banner"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"os"
	"path/filepath"
	"strings"
)

var (
	logger    *zap.Logger
	clusters  []*types.Cluster
	nginxConf = types.NewNginxConf(clusters)
	opts      *options.NginxConfGeneratorOptions
	ver       = version.Get()
)

func init() {
	opts = options.GetNginxConfGeneratorOptions()

	rootCmd.Flags().StringVarP(&opts.KubeConfigPaths, "kubeConfigPaths", "", filepath.Join(os.Getenv("HOME"), ".kube", "config"),
		"comma separated list of kubeconfig file paths to access with the cluster")
	rootCmd.Flags().StringVarP(&opts.WorkerNodeLabel, "workerNodeLabel", "", "worker",
		"label to specify worker nodes")
	rootCmd.Flags().StringVarP(&opts.CustomAnnotation, "customAnnotation", "", "nginx-conf-generator/enabled",
		"annotation to specify selectable services")
	rootCmd.Flags().StringVarP(&opts.TemplateInputFile, "templateInputFile", "", "resources/ncg.conf.tmpl",
		"path of the template input file to be able to render and print to --templateOutputFile")
	rootCmd.Flags().StringVarP(&opts.TemplateOutputFile, "templateOutputFile", "", "/etc/nginx/conf.d/ncg.conf",
		"path of the template output file which is a valid Nginx conf file")
	rootCmd.Flags().IntVarP(&opts.MetricsPort, "metricsPort", "", 5000,
		"port of the metrics server")
	rootCmd.Flags().IntVarP(&opts.WriteTimeoutSeconds, "writeTimeoutSeconds", "", 10,
		"write timeout of the metrics server")
	rootCmd.Flags().IntVarP(&opts.ReadTimeoutSeconds, "readTimeoutSeconds", "", 10,
		"read timeout of the metrics server")
	rootCmd.Flags().StringVarP(&opts.BannerFilePath, "bannerFilePath", "", "build/ci/banner.txt",
		"relative path of the banner file")
	rootCmd.Flags().StringVarP(&opts.MetricsEndpoint, "metricsEndpoint", "", "/metrics",
		"endpoint to provide prometheus metrics")
	rootCmd.Flags().BoolVarP(&opts.VerboseLog, "verbose", "v", false, "verbose output of the logging library (default false)")

	if err := rootCmd.Flags().MarkHidden("bannerFilePath"); err != nil {
		panic("fatal error occured while hiding flag")
	}

	logger = logging.GetLogger()
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "nginx-conf-generator",
	Short:   "A fancy tool which manages Nginx configuration according to Kubernetes NodePort type services",
	Version: ver.GitVersion,
	Long: `nginx-conf-generator gets the port of NodePort type services which contains specific annotation. Then modifies
the Nginx configuration and reloads the Nginx process. nginx-conf-generator can also work with multiple Kubernetes clusters.
This means you can route traffic to multiple Kubernetes clusters through a Nginx server for your NodePort type services`,
	Run: func(cmd *cobra.Command, args []string) {
		if opts.VerboseLog {
			logging.Atomic.SetLevel(zap.DebugLevel)
		}

		if _, err := os.Stat(opts.BannerFilePath); err == nil {
			bannerBytes, _ := os.ReadFile(opts.BannerFilePath)
			banner.Init(os.Stdout, true, false, strings.NewReader(string(bannerBytes)))
		}

		logger.Info("nginx-conf-generator is started",
			zap.String("appVersion", ver.GitVersion),
			zap.String("goVersion", ver.GoVersion),
			zap.String("goOS", ver.GoOs),
			zap.String("goArch", ver.GoArch),
			zap.String("gitCommit", ver.GitCommit),
			zap.String("buildDate", ver.BuildDate))

		kubeConfigPathArr := strings.Split(opts.KubeConfigPaths, ",")
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
			logger.With(zap.String("masterIP", cluster.MasterIP))

			informers.RunNodeInformer(cluster, clientSet, logger, nginxConf)
			informers.RunServiceInformer(cluster, clientSet, logger, nginxConf)
		}

		go func() {
			if err := metrics.RunMetricsServer(); err != nil {
				logger.Fatal("fatal error occurred while spinning up metrics server",
					zap.String("error", err.Error()))
			}
		}()

		select {}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
