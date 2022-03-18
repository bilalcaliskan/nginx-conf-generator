package informers

import (
	"nginx-conf-generator/internal/k8s/types"
	"nginx-conf-generator/internal/options"
	"strconv"
	"time"

	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// RunServiceInformer spins up a shared informer factory and fetch Kubernetes service events
func RunServiceInformer(cluster *types.Cluster, clientSet kubernetes.Interface, logger *zap.Logger,
	ncgo *options.NginxConfGeneratorOptions, nginxConf *types.NginxConf) {

	informerFactory := informers.NewSharedInformerFactory(clientSet, time.Second*30)
	serviceInformer := informerFactory.Core().V1().Services()
	serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			service := obj.(*v1.Service)
			var ok bool

			if service.Annotations[ncgo.CustomAnnotation] == "true" {
				ok = true
			}

			if service.Spec.Type == v1.ServiceTypeNodePort && ok && service.Annotations[ncgo.CustomAnnotation] == "true" {
				logger.Info("valid service added", zap.String("name", service.Name),
					zap.String("namespace", service.Namespace), zap.Int32("nodePort", service.Spec.Ports[0].NodePort))

				nodePort := types.NewNodePort(cluster.MasterIP, service.Spec.Ports[0].NodePort)
				_, found := findNodePort(cluster.NodePorts, nodePort)
				if !found {
					logger.Info("adding nodePort to backend.NodePorts", zap.Int32("nodePort", nodePort.Port))
					addNodePort(&cluster.NodePorts, nodePort)
				}

				// Apply changes to the template
				ncgo.Mu.Lock()
				err := renderTemplate(ncgo.TemplateInputFile, ncgo.TemplateOutputFile, nginxConf)
				ncgo.Mu.Unlock()
				if err != nil {
					logger.Fatal(ErrRenderTemplate, zap.Error(err))
				}

				// reload Nginx service
				err = reloadNginx()
				if err != nil {
					logger.Fatal(ErrReloadNginx, zap.String("error", err.Error()))
				} else {
					logger.Info(SuccessNginxReload)
				}
			} else {
				logger.Info("service is not valid, it is not annotated or not a nodePort type service",
					zap.String("name", service.Name),
					zap.String("namespace", service.Namespace))
			}
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			oldService := oldObj.(*v1.Service)
			newService := newObj.(*v1.Service)
			// There is an actual update on the service
			oldOk, _ := strconv.ParseBool(oldService.Annotations[ncgo.CustomAnnotation])
			newOk, _ := strconv.ParseBool(newService.Annotations[ncgo.CustomAnnotation])

			if oldService.ResourceVersion != newService.ResourceVersion {
				if oldOk && oldService.Spec.Type == v1.ServiceTypeNodePort {
					if newOk && newService.Spec.Type == v1.ServiceTypeNodePort {
						oldNodePort := types.NewNodePort(cluster.MasterIP, oldService.Spec.Ports[0].NodePort)
						newNodePort := types.NewNodePort(cluster.MasterIP, newService.Spec.Ports[0].NodePort)
						if !oldNodePort.Equals(newNodePort) {
							logger.Info("nodePort changed on the valid service, updating",
								zap.String("masterIP", cluster.MasterIP), zap.String("name", oldService.Name),
								zap.String("namespace", oldService.Namespace), zap.Int32("oldNodePort", oldNodePort.Port),
								zap.Int32("newNodePort", newNodePort.Port))
							updateNodePort(&cluster.NodePorts, cluster.Workers, oldNodePort, newNodePort)
						} else {
							logger.Info("nodePort ports are the same on the updated service, nothing to do",
								zap.String("masterIP", cluster.MasterIP), zap.String("name", oldService.Name),
								zap.String("namespace", oldService.Namespace), zap.Int32("oldNodePort", oldNodePort.Port),
								zap.Int32("newNodePort", newNodePort.Port))
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
					}
				}

				// Apply changes to the template
				ncgo.Mu.Lock()
				err := renderTemplate(ncgo.TemplateInputFile, ncgo.TemplateOutputFile, nginxConf)
				ncgo.Mu.Unlock()
				if err != nil {
					logger.Fatal(ErrRenderTemplate, zap.Error(err))
				}

				// reload Nginx service
				err = reloadNginx()
				if err != nil {
					logger.Fatal(ErrReloadNginx, zap.String("error", err.Error()))
				} else {
					logger.Info(SuccessNginxReload)
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			service := obj.(*v1.Service)
			var ok bool
			if service.Annotations[ncgo.CustomAnnotation] == "true" {
				ok = true
			}

			if service.Spec.Type == v1.ServiceTypeNodePort && ok {
				nodePort := types.NewNodePort(cluster.MasterIP, service.Spec.Ports[0].NodePort)
				index, found := findNodePort(cluster.NodePorts, nodePort)
				if found {
					logger.Info("valid service deleted, removing from cluster.NodePorts",
						zap.String("name", service.Name), zap.String("namespace", service.Namespace))
					removeNodePort(&cluster.NodePorts, index)
					logger.Info("successfully removed deleted service from cluster.NodePorts",
						zap.String("name", service.Name), zap.String("namespace", service.Namespace))
				}
			}

			// Apply changes to the template
			ncgo.Mu.Lock()
			err := renderTemplate(ncgo.TemplateInputFile, ncgo.TemplateOutputFile, nginxConf)
			ncgo.Mu.Unlock()
			if err != nil {
				logger.Fatal(ErrRenderTemplate, zap.Error(err))
			}

			// reload Nginx service
			err = reloadNginx()
			if err != nil {
				logger.Fatal(ErrReloadNginx, zap.String("error", err.Error()))
			} else {
				logger.Info(SuccessNginxReload)
			}
		},
	})
	informerFactory.Start(wait.NeverStop)
	informerFactory.WaitForCacheSync(wait.NeverStop)
}
