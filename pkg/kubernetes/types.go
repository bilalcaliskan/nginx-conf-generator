package kubernetes

import (
	v1 "k8s.io/api/core/v1"
)

type NginxConf struct {
	Clusters []*Cluster
}

type Cluster struct {
	MasterIP string
	Workers []*Worker
	NodePorts []*nodePort
}

type nodePort struct {
	MasterIP string
	Port int32
	Workers []*Worker
}

type Worker struct {
	MasterIP, HostIP string
	NodeCondition v1.ConditionStatus
}

func NewCluster(masterIP string, workers []*Worker) *Cluster {
	return &Cluster{
		MasterIP: masterIP,
		Workers: workers,
	}
}

func newNodePort(masterIP string, port int32) *nodePort {
	return &nodePort{
		MasterIP: masterIP,
		Workers: make([]*Worker, 0),
		Port:     port,
	}
}

func NewWorker(masterIp, hostIp string, nodeReady v1.ConditionStatus) *Worker {
	return &Worker{
		MasterIP: masterIp,
		HostIP: hostIp,
		NodeCondition: nodeReady,
	}
}

func (nodePort *nodePort) Equals(other *nodePort) bool {
	isMasterIPEquals := nodePort.MasterIP == other.MasterIP
	isPortEquals := nodePort.Port == other.Port
	return isMasterIPEquals && isPortEquals
}

func (worker *Worker) Equals(other *Worker) bool {
	isMasterIPEquals := worker.MasterIP == other.MasterIP
	isHostIPEquals := worker.HostIP == other.HostIP
	return isMasterIPEquals && isHostIPEquals
}