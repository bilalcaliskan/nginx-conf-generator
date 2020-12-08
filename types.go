package main

import (
	v1 "k8s.io/api/core/v1"
)

/*func (backend *Backend) Equals(other *Backend) bool {
	isMasterIPEquals := backend.MasterIP == other.MasterIP
	isWorkersEqual := len(backend.Workers) == len(other.Workers)
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

type NginxConf struct {
	VServers []VServer
	Backends []Backend
}

type VServer struct {
	Port int32
}

func (vserver *VServer) Equals(other VServer) bool {
	return vserver.Port == other.Port
}

func newVServer(port int32) *VServer {
	return &VServer{
		Port:    port,
	}
}

type Backend struct {
	MasterIP string
	Workers []Worker
	VServers []VServer
}

func newBackend(masterIP string, workers []Worker, vservers []VServer) *Backend {
	return &Backend{
		MasterIP: masterIP,
		Workers: workers,
		VServers: vservers,
	}
}

func (backend *Backend) isVServerExists(vserver VServer) bool {
	for _, v := range backend.VServers {
		if v.Equals(vserver) {
			return true
		}
	}
	return false
}

type Worker struct {
	MasterIP, HostIP string
	NodeCondition v1.ConditionStatus
}

func (worker *Worker) Equals(other *Worker) bool {
	isMasterIPEquals := worker.MasterIP == other.MasterIP
	isHostIPEquals := worker.HostIP == other.HostIP
	return isMasterIPEquals && isHostIPEquals
}

func newWorker(masterIp, hostIp string, nodeReady v1.ConditionStatus) *Worker {
	return &Worker{
		MasterIP: masterIp,
		HostIP: hostIp,
		NodeCondition: nodeReady,
	}
}