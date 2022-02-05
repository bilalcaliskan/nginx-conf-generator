package types

// NginxConf is the biggest struct in app, keeps track of k8s clusters
type NginxConf struct {
	Clusters []*Cluster
}

// NewNginxConf generates a NginxConf struct with specified fields
func NewNginxConf(cluster []*Cluster) *NginxConf {
	return &NginxConf{
		Clusters: cluster,
	}
}
