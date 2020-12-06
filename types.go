package main

type NginxConf struct {
	VServers []VServer
	Backends []Backend
}

type VServer struct {
	Port int32
	Backend Backend
}

/*func (vserver *VServer) Equals(other *VServer) bool {
	isPortEquals := vserver.Port == other.Port
	isBackendEquals := vserver.Backend.Equals(&other.Backend)
	return isPortEquals && isBackendEquals
}*/

func newVServer(port int32, backend Backend) *VServer {
	return &VServer{
		Port:    port,
		Backend: backend,
	}
}

type Backend struct {
	MasterIP string
	Workers []Worker
	NodePorts []int32
}

/*func (backend *Backend) Equals(other *Backend) bool {
	isNameEquals := backend.Name == other.Name
	isIpEquals := backend.IP == other.IP
	isPortEquals := backend.Port == other.Port
	/*isWorkersEqual := len(backend.Workers) == len(other.Workers)
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
}

func newBackend(name, masterIp string, nodePort int32) *Backend {
	return &Backend{
		Name:    name,
		IP:      masterIp,
		Port:    nodePort,
		Workers: make([]Worker, 0),
	}
}*/

type Worker struct {
	IP string
}

func newWorker(ip string) *Worker {
	return &Worker{
		IP:    ip,
	}
}