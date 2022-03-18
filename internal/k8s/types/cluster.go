package types

import "sync"

// Cluster is the logical representation of k8s clusters
type Cluster struct {
	MasterIP  string
	Workers   []*Worker
	NodePorts []*NodePort
	Mu        sync.Mutex
}

// NewCluster creates a Cluster struct with specified parameters and returns it
func NewCluster(masterIP string, workers []*Worker) *Cluster {
	return &Cluster{
		MasterIP: masterIP,
		Workers:  workers,
	}
}
