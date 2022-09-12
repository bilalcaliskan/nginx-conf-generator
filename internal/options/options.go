package options

import (
	"sync"
)

var nginxConfGeneratorOptions = &NginxConfGeneratorOptions{}

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
	// BannerFilePath is the relative path to the banner file
	BannerFilePath string
	// VerboseLog is the verbosity of the logging library
	VerboseLog bool
	Mu         sync.Mutex
}

// GetNginxConfGeneratorOptions returns the pointer of NginxConfGeneratorOptions
func GetNginxConfGeneratorOptions() *NginxConfGeneratorOptions {
	return nginxConfGeneratorOptions
}
