package main

import (
	v1 "k8s.io/api/core/v1"
)

type NginxConf struct {
	Clusters []*Cluster
}

type Cluster struct {
	MasterIP string
	Workers []*Worker
	NodePorts []*NodePort
}

type NodePort struct {
	MasterIP string
	Port int32
	Workers []*Worker
}

type Worker struct {
	MasterIP, HostIP string
	NodeCondition v1.ConditionStatus
}

func newCluster(masterIP string, workers []*Worker) *Cluster {
	return &Cluster{
		MasterIP: masterIP,
		Workers: workers,
	}
}

func newNodePort(masterIP string, port int32) *NodePort {
	return &NodePort{
		MasterIP: masterIP,
		Workers: make([]*Worker, 0),
		Port:     port,
	}
}

func newWorker(masterIp, hostIp string, nodeReady v1.ConditionStatus) *Worker {
	return &Worker{
		MasterIP: masterIp,
		HostIP: hostIp,
		NodeCondition: nodeReady,
	}
}

func (nodePort *NodePort) Equals(other *NodePort) bool {
	isMasterIPEquals := nodePort.MasterIP == other.MasterIP
	isPortEquals := nodePort.Port == other.Port
	return isMasterIPEquals && isPortEquals
}

func (worker *Worker) Equals(other *Worker) bool {
	isMasterIPEquals := worker.MasterIP == other.MasterIP
	isHostIPEquals := worker.HostIP == other.HostIP
	return isMasterIPEquals && isHostIPEquals
}

/*func (cluster *Cluster) Equals(other *Cluster) bool {
	isMasterIPEquals := cluster.MasterIP == other.MasterIP
	isWorkersEqual := len(cluster.Workers) == len(other.Workers)
	if isWorkersEqual {  // copy slices so sorting won't affect original structs
		backendWorkers := make([]Worker, len(backend.Workers))
		otherWorkers := make([]Worker, len(other.Workers))
		copy(backend.Workers, backendWorkers)
		copy(other.Workers, otherWorkers)
		// Sort by index, keeping original order or equal elements.
		sort.SliceStable(backendWorkers, func(i, j int) bool {
			return backendWorkers[i].Index < backendWorkers[j].Index
		})
		sort.SliceStable(otherWorkers, func(i, j int) bool {
			return otherWorkers[i].Index < otherWorkers[j].Index
		})
		for index, item := range backendWorkers {
			if item != otherWorkers[index] {
				isWorkersEqual = false
			}
		}
	}
	// return isNameEquals && isIpEquals && isPortEquals && isWorkersEqual
	return isNameEquals && isIpEquals && isPortEquals
}*/