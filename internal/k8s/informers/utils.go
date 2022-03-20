package informers

import (
	"fmt"
	"html/template"
	"nginx-conf-generator/internal/k8s/types"
	"nginx-conf-generator/internal/options"
	"os"
	"os/exec"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func addWorkerToNodePorts(nodePorts []*types.NodePort, worker *types.Worker) {
	for _, v := range nodePorts {
		_, found := findWorker(v.Workers, worker)
		if !found {
			v.Workers = append(v.Workers, worker)
		}
	}
}

func removeWorkerFromNodePorts(nodePorts []*types.NodePort, worker *types.Worker) {
	for _, v := range nodePorts {
		index, found := findWorker(v.Workers, worker)
		if found {
			v.Workers = append((v.Workers)[:index], (v.Workers)[index+1:]...)
		}
	}
}

func addWorkersToNodePort(workers []*types.Worker, nodePort *types.NodePort) {
	for _, v := range workers {
		_, found := findWorker(nodePort.Workers, v)
		if !found {
			nodePort.Workers = append(nodePort.Workers, v)
		}
	}
}

func addWorker(cluster *types.Cluster, worker *types.Worker) {
	_, found := findWorker(cluster.Workers, worker)
	if !found {
		cluster.Mu.Lock()
		cluster.Workers = append(cluster.Workers, worker)
		cluster.Mu.Unlock()
	}
}

func removeWorkerFromCluster(cluster *types.Cluster, index int) {
	cluster.Mu.Lock()
	cluster.Workers = append((cluster.Workers)[:index], (cluster.Workers)[index+1:]...)
	cluster.Mu.Unlock()
}

func findWorker(workers []*types.Worker, worker *types.Worker) (int, bool) {
	for i, item := range workers {
		if worker.Equals(item) {
			return i, true
		}
	}
	return -1, false
}

func addNodePort(nodePorts *[]*types.NodePort, nodePort *types.NodePort) {
	_, found := findNodePort(*nodePorts, nodePort)
	if !found {
		*nodePorts = append(*nodePorts, nodePort)
	}
}

func findNodePort(nodePorts []*types.NodePort, nodePort *types.NodePort) (int, bool) {
	for i, item := range nodePorts {
		if nodePort.Equals(item) {
			return i, true
		}
	}
	return -1, false
}

func removeNodePort(nodePorts *[]*types.NodePort, index int) {
	*nodePorts = append((*nodePorts)[:index], (*nodePorts)[index+1:]...)
}

func updateNodePort(nodePorts *[]*types.NodePort, workers []*types.Worker, oldNodePort *types.NodePort, newNodePort *types.NodePort) {
	oldIndex, oldFound := findNodePort(*nodePorts, oldNodePort)
	if oldFound {
		removeNodePort(nodePorts, oldIndex)
	}
	addWorkersToNodePort(workers, newNodePort)
	addNodePort(nodePorts, newNodePort)
}

// GetConfig creates a rest.Config and returns it
func GetConfig(kubeConfigPath string) (*rest.Config, error) {
	var config *rest.Config
	var err error

	config, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// GetClientSet creates a kubernetes.Clientset and returns it
func GetClientSet(config *rest.Config) (*kubernetes.Clientset, error) {
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientSet, nil
}

func isNodeReady(node *v1.Node) v1.ConditionStatus {
	for _, v := range node.Status.Conditions {
		if v.Type == v1.NodeReady {
			return v.Status
		}
	}
	return v1.ConditionFalse
}

func reloadNginx() error {
	cmd := exec.Command("nginx", "-s", "reload")
	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func applyChanges(ncgo *options.NginxConfGeneratorOptions, conf *types.NginxConf) error {
	// Apply changes to the template
	ncgo.Mu.Lock()
	if err := renderTemplate(ncgo.TemplateInputFile, ncgo.TemplateOutputFile, conf); err != nil {
		return fmt.Errorf("%s, %s", ErrRenderTemplate, err.Error())
	}
	ncgo.Mu.Unlock()

	// Reload Nginx service
	if err := reloadNginx(); err != nil {
		return fmt.Errorf("%s, %s", ErrReloadNginx, err.Error())
	}

	return nil
}

func renderTemplate(templateInputFile, templateOutputFile string, data interface{}) error {
	tpl := template.Must(template.ParseFiles(templateInputFile))
	f, err := os.Create(templateOutputFile)
	if err != nil {
		return err
	}

	err = tpl.ExecuteTemplate(f, "main", data)
	if err != nil {
		return err
	}

	err = f.Close()
	if err != nil {
		return err
	}

	return nil
}
