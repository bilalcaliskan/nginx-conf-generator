package main

type Backend struct {
	Name, IP string
	Index int
}

type VServer struct {
	Port int32
	Backend Backend
}

type NginxConf struct {
	VServers []VServer
	Backends []Backend
}