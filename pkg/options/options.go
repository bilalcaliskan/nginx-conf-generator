package options

import (
	"github.com/spf13/pflag"
	"os"
	"path/filepath"
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
}

// GetNginxConfGeneratorOptions returns the pointer of NginxConfGeneratorOptions
func GetNginxConfGeneratorOptions() *NginxConfGeneratorOptions {
	return nginxConfGeneratorOptions
}

func (ncgo *NginxConfGeneratorOptions) addFlags(flag *pflag.FlagSet) {
	flag.StringVar(&ncgo.KubeConfigPaths, "kubeConfigPaths", filepath.Join(os.Getenv("HOME"), ".kube", "minikubeconfig"),
		"comma separated list of kubeconfig file paths to access with the cluster")
	flag.StringVar(&ncgo.WorkerNodeLabel, "workerNodeLabel", "node-role.k8s.io/worker", "label to specify "+
		"worker nodes, defaults to node-role.k8s.io/worker=")
	flag.StringVar(&ncgo.CustomAnnotation, "customAnnotation", "nginx-conf-generator/enabled", "annotation to specify "+
		"selectable services")
	flag.StringVar(&ncgo.TemplateInputFile, "templateInputFile", "resources/default.conf.tmpl", "input "+
		"path of the template file")
	flag.StringVar(&ncgo.TemplateOutputFile, "templateOutputFile", "/etc/nginx/sites-enabled/default", "output "+
		"path of the template file")
}
