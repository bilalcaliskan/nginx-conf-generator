package main

type Backend struct {
	Name, IP string
	Port int32
}

type VServer struct {
	Port int32
	Backend Backend
}

type NginxConf struct {
	VServers []VServer
	Backends []Backend
}