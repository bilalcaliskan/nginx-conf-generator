package informers

import (
	"context"
	"nginx-conf-generator/internal/k8s/types"
	"nginx-conf-generator/internal/logging"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (fAPI *FakeAPI) getService(name string) (*v1.Service, error) {
	ctx, cancel := context.WithTimeout(parentCtx, 60*time.Second)
	defer cancel()
	return fAPI.ClientSet.CoreV1().Services(fAPI.Namespace).Get(ctx, name, metav1.GetOptions{})
}

func (fAPI *FakeAPI) createService(name string, serviceType v1.ServiceType, annotationEnabled bool) (*v1.Service, error) {
	service := &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app": name,
			},
			Annotations: map[string]string{
				opts.CustomAnnotation: strconv.FormatBool(annotationEnabled),
			},
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name:     "http",
					Protocol: v1.ProtocolTCP,
					Port:     8080,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 8080,
						StrVal: "",
					},
				},
			},
			Selector: map[string]string{
				"app": name,
			},
			Type: serviceType,
		},
	}

	ctx, cancel := context.WithTimeout(parentCtx, 60*time.Second)
	defer cancel()
	node, err := fAPI.ClientSet.CoreV1().Services(fAPI.Namespace).Create(ctx, service, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	return node, nil
}

func (fAPI *FakeAPI) updateService(name string, annotationEnabled, updateNodeport bool, serviceType v1.ServiceType, version string) (*v1.Service, error) {
	var err error
	service, _ := fAPI.getService(name)
	service.Annotations[opts.CustomAnnotation] = strconv.FormatBool(annotationEnabled)
	service.ResourceVersion = version
	if updateNodeport && serviceType == v1.ServiceTypeNodePort {
		service.Spec.Ports[0].NodePort = service.Spec.Ports[0].NodePort + 1
	}
	service.Spec.Type = serviceType

	ctx, cancel := context.WithTimeout(parentCtx, 60*time.Second)
	defer cancel()
	service, err = fAPI.ClientSet.CoreV1().Services(fAPI.Namespace).Update(ctx, service, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}

	return service, nil
}

func (fAPI *FakeAPI) deleteService(name string) error {
	ctx, cancel := context.WithTimeout(parentCtx, 60*time.Second)
	defer cancel()
	return fAPI.ClientSet.CoreV1().Services(fAPI.Namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

func TestRunServiceInformer(t *testing.T) {
	api := getFakeAPI()
	assert.NotNil(t, api)

	opts.Mu.Lock()
	opts.TemplateInputFile = "../../../resources/ncg.conf.tmpl"
	opts.Mu.Unlock()

	var clusters []*types.Cluster
	nginxConf := types.NewNginxConf(clusters)
	cluster := types.NewCluster("", make([]*types.Worker, 0))
	nginxConf.Clusters = append(nginxConf.Clusters, cluster)
	t.Logf(opts.CustomAnnotation)

	go func() {
		RunServiceInformer(cluster, api.ClientSet, logging.NewLogger(), nginxConf)
	}()

	cases := []struct {
		caseName                                                             string
		serviceTypeOnCreate, serviceTypeOnUpdate                             v1.ServiceType
		annotationEnabledOnCreate, annotationEnabledOnUpdate, updateNodeport bool
	}{
		{"case1", v1.ServiceTypeNodePort, v1.ServiceTypeNodePort,
			true, true, false},
		{"case2", v1.ServiceTypeNodePort, v1.ServiceTypeNodePort,
			true, true, true},
		{"case3", v1.ServiceTypeClusterIP, v1.ServiceTypeNodePort,
			true, false, false},
		{"case4", v1.ServiceTypeNodePort, v1.ServiceTypeNodePort,
			true, false, false},

		{"case5", v1.ServiceTypeClusterIP, v1.ServiceTypeNodePort,
			false, true, false},
	}

	for _, tc := range cases {
		t.Run(tc.caseName, func(t *testing.T) {
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				time.Sleep(2 * time.Second)
				defer wg.Done()
				service, err := api.createService("nginx-a", tc.serviceTypeOnCreate, tc.annotationEnabledOnCreate)
				assert.Nil(t, err)
				assert.NotNil(t, service)
			}()
			wg.Wait()

			wg.Add(1)
			go func() {
				time.Sleep(2 * time.Second)
				defer wg.Done()
				service, err := api.updateService("nginx-a", tc.annotationEnabledOnUpdate, tc.updateNodeport,
					tc.serviceTypeOnUpdate, "123456")
				assert.NotNil(t, service)
				assert.Nil(t, err)
			}()
			wg.Wait()

			wg.Add(1)
			go func() {
				time.Sleep(2 * time.Second)
				defer wg.Done()
				service, err := api.getService("nginx-a")
				assert.Nil(t, err)
				assert.NotNil(t, service)
				assert.Equal(t, strconv.FormatBool(tc.annotationEnabledOnUpdate), service.Annotations[opts.CustomAnnotation])
			}()
			wg.Wait()

			wg.Add(1)
			go func() {
				time.Sleep(2 * time.Second)
				defer wg.Done()
				err := api.deleteService("nginx-a")
				assert.Nil(t, err)
			}()
			wg.Wait()

			wg.Add(1)
			go func() {
				time.Sleep(2 * time.Second)
				defer wg.Done()
				service, err := api.getService("nginx-a")
				assert.NotNil(t, err)
				assert.Nil(t, service)
			}()
			wg.Wait()
		})
	}
}
