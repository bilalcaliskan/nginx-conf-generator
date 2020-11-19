package main

import v1 "k8s.io/api/core/v1"

type Generic struct {

}

type Backend struct {
	Name, IP string
	Port int32
	Generic
}

type VServer struct {
	Port int32
	Backend Backend
	Generic
}

type NginxConf struct {
	VServers []VServer
	Backends []Backend
	Generic
}

type K8sService struct {
	Namespace string
	Name string
	NodePort int32
	Type v1.ServiceType
	Generic
}