package options

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/pflag"
)

var nginxConfGeneratorOptions = &NginxConfGeneratorOptions{}

func init() {
	nginxConfGeneratorOptions.addFlags(pflag.CommandLine)
	pflag.Parse()
}

// NginxConfGeneratorOptions contains frequent command line and application options.
type NginxConfGeneratorOptions struct {
	// KubeConfigPaths is the comma separated list of kubeconfig file paths to access with the cluster
	KubeConfigPaths string
	// WorkerNodeLabel is the label to specify worker nodes, defaults to node-role.k8s.io/worker=
	WorkerNodeLabel string
	// CustomAnnotation is the annotation to specify selectable services
	CustomAnnotation string
	// TemplateInputFile is the input path of the template file
	TemplateInputFile string
	// TemplateOutputFile is the output path of the template file
	TemplateOutputFile string
	// MetricsPort is the port of the metric server to expose prometheus metrics
	MetricsPort int
	// WriteTimeoutSeconds is the timeout of the write timeout for metrics server
	WriteTimeoutSeconds int
	// ReadTimeoutSeconds is the timeout of the write timeout for metrics server
	ReadTimeoutSeconds int
	// MetricsEndpoint is the endpoint to consume prometheus metrics
	MetricsEndpoint string
	Mu              sync.Mutex
}

// GetNginxConfGeneratorOptions returns the pointer of NginxConfGeneratorOptions
func GetNginxConfGeneratorOptions() *NginxConfGeneratorOptions {
	return nginxConfGeneratorOptions
}

func (ncgo *NginxConfGeneratorOptions) addFlags(flag *pflag.FlagSet) {
	// filepath.Join(os.Getenv("HOME")
	flag.StringVar(&ncgo.KubeConfigPaths, "kubeConfigPaths", filepath.Join(os.Getenv("HOME"), ".kube", "config"),
		"comma separated list of kubeconfig file paths to access with the cluster")
	flag.StringVar(&ncgo.WorkerNodeLabel, "workerNodeLabel", "worker", "label to specify "+
		"worker nodes, defaults to worker")
	flag.StringVar(&ncgo.CustomAnnotation, "customAnnotation", "nginx-conf-generator/enabled", "annotation to specify "+
		"selectable services")
	flag.StringVar(&ncgo.TemplateInputFile, "templateInputFile", "resources/default.conf.tmpl", "input "+
		"path of the template file")
	flag.StringVar(&ncgo.TemplateOutputFile, "templateOutputFile", "/etc/nginx/conf.d/default", "output "+
		"path of the template file")
	flag.IntVar(&ncgo.MetricsPort, "metricsPort", 5000, "port of the metrics server")
	flag.IntVar(&ncgo.WriteTimeoutSeconds, "writeTimeoutSeconds", 10, "write timeout of the metrics server")
	flag.IntVar(&ncgo.ReadTimeoutSeconds, "readTimeoutSeconds", 10, "read timeout of the metrics server")
	flag.StringVar(&ncgo.MetricsEndpoint, "metricsEndpoint", "/metrics", "endpoint to provide prometheus metrics")
}
