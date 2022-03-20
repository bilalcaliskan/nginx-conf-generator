package informers

import (
	"nginx-conf-generator/internal/k8s/types"
	"nginx-conf-generator/internal/options"
	"time"

	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// RunServiceInformer spins up a shared informer factory and fetch Kubernetes service events
func RunServiceInformer(cluster *types.Cluster, clientSet kubernetes.Interface, logger *zap.Logger, nginxConf *types.NginxConf) {
	ncgo := options.GetNginxConfGeneratorOptions()
	informerFactory := informers.NewSharedInformerFactory(clientSet, time.Second*30)
	serviceInformer := informerFactory.Core().V1().Services()
	serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if len(cluster.Workers) == 0 {
				logger.Warn("length of cluster.Workers is 0, can not add a server without any upstream server")
				return
			}

			service := obj.(*v1.Service)
			if val, ok := service.Annotations[ncgo.CustomAnnotation]; !ok || val != "true" {
				logger.Debug("service is not properly annotated, skipping...")
				return
			}

			if service.Spec.Type != v1.ServiceTypeNodePort {
				logger.Debug("not a NodePort type service, skipping...")
				return
			}

			logger.Info("valid service added", zap.String("name", service.Name),
				zap.String("namespace", service.Namespace), zap.Int32("nodePort", service.Spec.Ports[0].NodePort))

			nodePort := types.NewNodePort(cluster.MasterIP, service.Spec.Ports[0].NodePort)
			_, found := findNodePort(cluster.NodePorts, nodePort)
			if !found {
				logger.Info("adding nodePort to backend.NodePorts", zap.Int32("nodePort", nodePort.Port))
				addNodePort(&cluster.NodePorts, nodePort)
				addWorkersToNodePort(cluster.Workers, nodePort)
			}

			if err := applyChanges(ncgo, nginxConf); err != nil {
				logger.Fatal("fatal error occured while applying changes", zap.String("error", err.Error()))
			}
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			// TODO: inspect buggy cases
			if len(cluster.Workers) == 0 {
				logger.Warn("length of cluster.Workers is 0, can not add a server without any upstream server")
				return
			}

			var applyRequired bool

			oldService := oldObj.(*v1.Service)
			newService := newObj.(*v1.Service)
			oldVal, oldOk := oldService.Annotations[ncgo.CustomAnnotation]
			oldOk = oldOk && oldVal == "true"
			newVal, newOk := newService.Annotations[ncgo.CustomAnnotation]
			newOk = newOk && newVal == "true"

			// check if it's a real update
			if oldService.ResourceVersion == newService.ResourceVersion {
				logger.Debug("not a real update, skipping")
				return
			}

			if oldOk && oldService.Spec.Type == v1.ServiceTypeNodePort {
				if newOk && newService.Spec.Type == v1.ServiceTypeNodePort {
					if newService.Spec.Ports[0].NodePort != oldService.Spec.Ports[0].NodePort {
						oldNodePort := types.NewNodePort(cluster.MasterIP, oldService.Spec.Ports[0].NodePort)
						newNodePort := types.NewNodePort(cluster.MasterIP, newService.Spec.Ports[0].NodePort)
						logger.Info("nodePort changed on the valid service, updating",
							zap.String("masterIP", cluster.MasterIP), zap.String("name", oldService.Name),
							zap.String("namespace", oldService.Namespace), zap.Int32("oldNodePort", oldNodePort.Port),
							zap.Int32("newNodePort", newNodePort.Port))
						updateNodePort(&cluster.NodePorts, cluster.Workers, oldNodePort, newNodePort)
						applyRequired = true
					} else {
						logger.Info("nodePort ports are the same on the updated service, nothing to do",
							zap.String("masterIP", cluster.MasterIP), zap.String("name", oldService.Name),
							zap.String("namespace", oldService.Namespace))
					}
				} else {
					oldNodePort := types.NewNodePort(cluster.MasterIP, oldService.Spec.Ports[0].NodePort)
					oldIndex, oldFound := findNodePort(cluster.NodePorts, oldNodePort)
					if oldFound {
						logger.Info("removing service from cluster.NodePorts because it is no more nodePort "+
							"type or no more labelled!", zap.String("masterIP", cluster.MasterIP),
							zap.String("name", oldService.Name), zap.String("namespace", oldService.Namespace))
						removeNodePort(&cluster.NodePorts, oldIndex)
						logger.Info("successfully removed service from cluster.NodePorts",
							zap.String("name", oldService.Name),
							zap.String("namespace", oldService.Namespace))
						applyRequired = true
					}
				}
			} else if newOk && newService.Spec.Type == v1.ServiceTypeNodePort {
				newNodePort := types.NewNodePort(cluster.MasterIP, newService.Spec.Ports[0].NodePort)
				_, newFound := findNodePort(cluster.NodePorts, newNodePort)
				if !newFound {
					logger.Info("adding service to cluster.NodePorts because it is labelled and nodePort "+
						"type service", zap.String("name", newService.Name),
						zap.String("namespace", newService.Namespace), zap.Int32("nodePort", newNodePort.Port))
					addNodePort(&cluster.NodePorts, newNodePort)
					addWorkersToNodePort(cluster.Workers, newNodePort)
					logger.Info("successfully added service to cluster.NodePorts",
						zap.String("name", newService.Name), zap.String("namespace", newService.Namespace),
						zap.Int32("nodePort", newNodePort.Port))
					applyRequired = true
				}
			}

			if applyRequired {
				if err := applyChanges(ncgo, nginxConf); err != nil {
					logger.Fatal("fatal error occured while applying changes", zap.String("error", err.Error()))
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			if len(cluster.Workers) == 0 {
				logger.Info("length of cluster.Workers is 0, can not add a server without any upstream server")
				return
			}

			service := obj.(*v1.Service)
			if val, ok := service.Annotations[ncgo.CustomAnnotation]; !ok || val != "true" {
				logger.Debug("service is not properly annotated, skipping...")
				return
			}

			if service.Spec.Type != v1.ServiceTypeNodePort {
				logger.Debug("not a NodePort type service, skipping...")
				return
			}

			nodePort := types.NewNodePort(cluster.MasterIP, service.Spec.Ports[0].NodePort)
			index, found := findNodePort(cluster.NodePorts, nodePort)
			if found {
				logger.Info("valid service deleted, removing from cluster.NodePorts",
					zap.String("name", service.Name), zap.String("namespace", service.Namespace))
				removeNodePort(&cluster.NodePorts, index)
				logger.Info("successfully removed deleted service from cluster.NodePorts",
					zap.String("name", service.Name), zap.String("namespace", service.Namespace))
			}

			if err := applyChanges(ncgo, nginxConf); err != nil {
				logger.Fatal("fatal error occured while applying changes", zap.String("error", err.Error()))
			}
		},
	})
	informerFactory.Start(wait.NeverStop)
	informerFactory.WaitForCacheSync(wait.NeverStop)
}
