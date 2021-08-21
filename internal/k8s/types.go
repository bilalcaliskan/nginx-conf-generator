package k8s

import (
	v1 "k8s.io/api/core/v1"
)

// NginxConf is the biggest struct in app, keeps track of k8s clusters
type NginxConf struct {
	Clusters []*Cluster
}

// Cluster is the logical representation of k8s clusters
type Cluster struct {
	MasterIP  string
	Workers   []*Worker
	NodePorts []*nodePort
}

type nodePort struct {
	MasterIP string
	Port     int32
	Workers  []*Worker
}

// Worker is the logical representation of the k8s worker nodes
type Worker struct {
	MasterIP, HostIP string
	NodeCondition    v1.ConditionStatus
}

// NewCluster creates a Cluster struct with specified parameters and returns it
func NewCluster(masterIP string, workers []*Worker) *Cluster {
	return &Cluster{
		MasterIP: masterIP,
		Workers:  workers,
	}
}

func newNodePort(masterIP string, port int32) *nodePort {
	return &nodePort{
		MasterIP: masterIP,
		Workers:  make([]*Worker, 0),
		Port:     port,
	}
}

// NewWorker creates a Worker struct with specified parameters and returns it
func NewWorker(masterIp, hostIp string, nodeReady v1.ConditionStatus) *Worker {
	return &Worker{
		MasterIP:      masterIp,
		HostIP:        hostIp,
		NodeCondition: nodeReady,
	}
}

// Equals method checks the equivalent of nodePort structs
func (nodePort *nodePort) Equals(other *nodePort) bool {
	isMasterIPEquals := nodePort.MasterIP == other.MasterIP
	isPortEquals := nodePort.Port == other.Port
	return isMasterIPEquals && isPortEquals
}

// Equals method checks the equivalent of Worker structs
func (worker *Worker) Equals(other *Worker) bool {
	isMasterIPEquals := worker.MasterIP == other.MasterIP
	isHostIPEquals := worker.HostIP == other.HostIP
	return isMasterIPEquals && isHostIPEquals
}
