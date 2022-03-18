package types

import "sync"

type NodePort struct {
	MasterIP string
	Port     int32
	Workers  []*Worker
	Mu       sync.Mutex
}

func NewNodePort(masterIP string, port int32) *NodePort {
	return &NodePort{
		MasterIP: masterIP,
		Port:     port,
	}
}

// Equals method checks the equivalent of nodePort structs
func (nodePort *NodePort) Equals(other *NodePort) bool {
	isMasterIPEquals := nodePort.MasterIP == other.MasterIP
	isPortEquals := nodePort.Port == other.Port
	return isMasterIPEquals && isPortEquals
}
