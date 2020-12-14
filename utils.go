package main

import (
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

/*func findVserver(vservers []VServer, vserver VServer) (int, bool) {
	for i, item := range vservers {
		if vserver.Equals(&item) {
			return i, true
		}
	}
	return -1, false
}

func findBackend(backends []Backend, backend Backend) (int, bool) {
	for i, item := range backends {
		if backend.Equals(&item) {
			return i, true
		}
	}
	return -1, false
}*/

func reloadNginx() error {
	cmd := exec.Command("nginx", "-s", "reload")
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func removeFromClustersSlice(slice []Cluster, index int) []Cluster {
	return append(slice[:index], slice[index+1:]...)
}

// TODO: refactor the function
/*func updateBackendsSlice(slice *[]Backend, oldBackend Backend, newBackend Backend) {
	oldIndex, oldFound := findBackend(*slice, oldBackend)
	if oldFound {
		log.Printf("update operation is starting on the nginxConf.Backends slice...%v\n", slice)
		log.Printf("removing backend %v from nginxConf.Backends slice!\n", oldBackend)
		*slice = removeFromBackendsSlice(*slice, oldIndex)

		_, newFound := findBackend(*slice, newBackend)
		if !newFound {
			log.Printf("adding backend %v to nginxConf.Backends slice!\n", newBackend)
			*slice = append(*slice, newBackend)
		} else {
			log.Printf("new backend %v already found in the nginxConf.Backends slice, skipping insertion...\n", newBackend)
		}
	} else {
		log.Printf("old backend %v not found in the nginxConf.Backends slice, skipping insertion, instead adding the new one %v...\n",
			oldBackend, newBackend)
		*slice = append(*slice, newBackend)
	}
	log.Printf("final nginxConf.Backends slice after update operation = %v\n", slice)
}

func removeFromVServersSlice(slice []VServer, index int) []VServer {
	return append(slice[:index], slice[index+1:]...)
}

// TODO: refactor the function
func updateVServersSlice(slice *[]VServer, oldVserver VServer, newVserver VServer) {
	oldIndex, oldFound := findVserver(*slice, oldVserver)
	if oldFound {
		log.Printf("update operation is starting on the nginxConf.VServers slice...%v\n", slice)
		log.Printf("removing backend %v from nginxConf.VServers slice!\n", oldVserver)
		*slice = removeFromVServersSlice(*slice, oldIndex)

		_, newFound := findVserver(*slice, newVserver)
		if !newFound {
			log.Printf("adding vserver %v to nginxConf.VServers slice!\n", newVserver)
			*slice = append(*slice, newVserver)
		} else {
			log.Printf("new vserver %v already found in the nginxConf.VServers slice, skipping insertion...\n", newVserver)
		}
	} else {
		log.Printf("old vserver %v not found in the nginxConf.VServers slice, skipping insertion, instead adding the new one %v...\n",
			oldVserver, newVserver)
		*slice = append(*slice, newVserver)
	}
	log.Printf("final nginxConf.VServers slice after update operation = %v\n", slice)
}

func addBackend(backends *[]Backend, backend Backend) {
	_, found := findBackend(*backends, backend)
	if !found {
		*backends = append(*backends, backend)
	}
}

func addVserver(vservers *[]VServer, vserver VServer) {
	_, found := findVserver(*vservers, vserver)
	if !found {
		*vservers = append(*vservers, vserver)
	}
}
*/

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