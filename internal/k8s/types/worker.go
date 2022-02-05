package types

import (
	v1 "k8s.io/api/core/v1"
)

// Worker is the logical representation of the k8s worker nodes
type Worker struct {
	MasterIP, HostIP string
	NodeCondition    v1.ConditionStatus
	NodePorts        []*NodePort
}

// NewWorker creates a Worker struct with specified parameters and returns it
func NewWorker(masterIp, hostIp string, nodeReady v1.ConditionStatus) *Worker {
	return &Worker{
		MasterIP:      masterIp,
		HostIP:        hostIp,
		NodeCondition: nodeReady,
	}
}

// Equals method checks the equivalent of Worker structs
func (worker *Worker) Equals(other *Worker) bool {
	isMasterIPEquals := worker.MasterIP == other.MasterIP
	isHostIPEquals := worker.HostIP == other.HostIP
	return isMasterIPEquals && isHostIPEquals
}

func (worker *Worker) HasNodePort(nodePort *NodePort) bool {
	for _, v := range worker.NodePorts {
		if v.Equals(nodePort) {
			return true
		}
	}
	return false
}
