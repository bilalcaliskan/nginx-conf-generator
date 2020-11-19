package main

import v1 "k8s.io/api/core/v1"

type Backend struct {
	Name, IP string
	Port int32
	K8sService
}

type VServer struct {
	Port int32
	Backend Backend
}

type NginxConf struct {
	VServers []VServer
	Backends []Backend
}

type K8sService struct {
	Namespace string
	Name string
	NodePort int32
	Type v1.ServiceType
}