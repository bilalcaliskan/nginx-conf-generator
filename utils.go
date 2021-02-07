package main

import (
	"go.uber.org/zap"
	"html/template"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"os"
	"os/exec"
)

func getConfig(kubeConfigPath string) (*rest.Config, error) {
	var config *rest.Config
	var err error

	config, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func getClientSet(config *rest.Config) (*kubernetes.Clientset, error) {
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientSet, nil
}

func checkError(err error) {
	if err != nil {

		log.Fatalln(err.Error())
	}
}

func reloadNginx(logger *zap.Logger) {
	logger.Info("trying to reload Nginx with rendered configuration file")
	cmd := exec.Command("nginx", "-s", "reload")
	err := cmd.Run()
	if err != nil {
		logger.Fatal("an error occured while reloading Nginx service", zap.String("error",
			err.Error()))
	} else {
		logger.Info("successfully reloaded Nginx service")
	}
}

func addWorker(slice *[]*Worker, worker *Worker) {
	_, found := findWorker(*slice, *worker)
	if !found {
		*slice = append(*slice, worker)
	}
}

func addNodePort(nodePorts *[]*NodePort, nodePort *NodePort) {
	_, found := findNodePort(*nodePorts, *nodePort)
	if !found {
		*nodePorts = append(*nodePorts, nodePort)
	}
}

func addWorkersToNodePort(workers []*Worker, nodePort *NodePort) {
	for _, v := range workers {
		_, found := findWorker(nodePort.Workers, *v)
		if !found {
			addWorker(&nodePort.Workers, v)
		}
	}
}

func addWorkerToNodePorts(nodePorts []*NodePort, worker *Worker) {
	for _, v := range nodePorts {
		_, found := findWorker(v.Workers, *worker)
		if !found {
			addWorker(&v.Workers, worker)
		}
	}
}

func removeWorkerFromNodePorts(nodePorts *[]*NodePort, worker *Worker) {
	for _, v := range *nodePorts {
		index, found := findWorker(v.Workers, *worker)
		if found {
			removeWorker(&v.Workers, index)
		}
	}
}

func findWorker(workers []*Worker, worker Worker) (int, bool) {
	for i, item := range workers {
		if worker.Equals(item) {
			return i, true
		}
	}
	return -1, false
}

func findNodePort(nodePorts []*NodePort, nodePort NodePort) (int, bool) {
	for i, item := range nodePorts {
		if nodePort.Equals(item) {
			return i, true
		}
	}
	return -1, false
}

func removeWorker(workers *[]*Worker, index int) {
	*workers = append((*workers)[:index], (*workers)[index+1:]...)
}

func removeNodePort(nodePorts *[]*NodePort, index int) {
	*nodePorts = append((*nodePorts)[:index], (*nodePorts)[index+1:]...)
}

func isNodeReady(node *v1.Node) v1.ConditionStatus {
	for _, v := range node.Status.Conditions {
		if v.Type == v1.NodeReady {
			return v.Status
		}
	}
	return "False"
}

func updateNodePort(nodePorts *[]*NodePort, workers []*Worker, oldNodePort *NodePort, newNodePort *NodePort) {
	oldIndex, oldFound := findNodePort(*nodePorts, *oldNodePort)
	if oldFound {
		removeNodePort(nodePorts, oldIndex)
	}
	addWorkersToNodePort(workers, newNodePort)
	addNodePort(nodePorts, newNodePort)
}

func renderTemplate(templateInputFile, templateOutputFile string, data interface{}) {
	// Apply changes to the template
	tpl := template.Must(template.ParseFiles(templateInputFile))
	f, err := os.Create(templateOutputFile)
	checkError(err)
	err = tpl.ExecuteTemplate(f, "main", data)
	checkError(err)
	err = f.Close()
	checkError(err)
}