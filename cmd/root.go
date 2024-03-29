package cmd

import (
	"github.com/bilalcaliskan/nginx-conf-generator/internal/k8s/informers"
	"github.com/bilalcaliskan/nginx-conf-generator/internal/k8s/types"
	"github.com/bilalcaliskan/nginx-conf-generator/internal/logging"
	"github.com/bilalcaliskan/nginx-conf-generator/internal/metrics"
	"github.com/bilalcaliskan/nginx-conf-generator/internal/options"
	"github.com/bilalcaliskan/nginx-conf-generator/internal/version"
	"github.com/dimiro1/banner"
	"github.com/pkg/errors"
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

	rootCmd.Flags().StringVarP(&opts.KubeConfigPaths, "kubeconfig-paths", "", filepath.Join(os.Getenv("HOME"), ".kube", "config"),
		"comma separated list of kubeconfig file paths to access with the cluster")
	rootCmd.Flags().StringVarP(&opts.WorkerNodeLabel, "worker-node-label", "", "worker",
		"label to specify worker nodes")
	rootCmd.Flags().StringVarP(&opts.CustomAnnotation, "custom-annotation", "", "nginx-conf-generator/enabled",
		"annotation to specify selectable services")
	rootCmd.Flags().StringVarP(&opts.TemplateInputFile, "template-input-file", "", "resources/ncg.conf.tmpl",
		"path of the template input file to be able to render and print to --template-output-file")
	rootCmd.Flags().StringVarP(&opts.TemplateOutputFile, "template-output-file", "", "/etc/nginx/conf.d/ncg.conf",
		"rendered output file path which is a valid Nginx conf file")
	rootCmd.Flags().IntVarP(&opts.MetricsPort, "metrics-port", "", 5000,
		"port of the metrics server")
	rootCmd.Flags().StringVarP(&opts.MetricsEndpoint, "metrics-endpoint", "", "/metrics",
		"endpoint to provide prometheus metrics")
	rootCmd.Flags().StringVarP(&opts.BannerFilePath, "banner-file-path", "", "build/ci/banner.txt",
		"relative path of the banner file")
	rootCmd.Flags().BoolVarP(&opts.VerboseLog, "verbose", "v", false, "verbose output of the logging library (default false)")

	if err := rootCmd.Flags().MarkHidden("banner-file-path"); err != nil {
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
	RunE: func(cmd *cobra.Command, args []string) error {
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
				logger.Error("an error occurred while getting k8s config", zap.String("error", err.Error()))
				return errors.Wrap(err, "unable to get rest config from k8s client")
			}

			clientSet, err := informers.GetClientSet(restConfig)
			if err != nil {
				logger.Error("an error occurred while getting clientset", zap.String("error", err.Error()))
				return errors.Wrap(err, "unable to get clientset from k8s client")
			}

			masterIp := strings.Split(strings.Split(restConfig.Host, "//")[1], ":")[0]
			cluster := types.NewCluster(masterIp, make([]*types.Worker, 0))
			nginxConf.Clusters = append(nginxConf.Clusters, cluster)
			logger.With(zap.String("masterIP", cluster.MasterIP))

			if err := informers.RunNodeInformer(cluster, clientSet, logger, nginxConf); err != nil {
				return err
			}

			if err := informers.RunServiceInformer(cluster, clientSet, logger, nginxConf); err != nil {
				return err
			}
		}

		go func() {
			if err := metrics.RunMetricsServer(); err != nil {
				logger.Fatal("an error occurred while spinning up metrics server",
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
