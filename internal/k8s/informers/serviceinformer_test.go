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

func (fAPI *FakeAPI) createService(name string, containsAnnotation bool) (*v1.Service, error) {
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
				opts.CustomAnnotation: strconv.FormatBool(containsAnnotation),
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
			Type: v1.ServiceTypeNodePort,
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

func (fAPI *FakeAPI) updateService(name string, annotationEnabled bool) (*v1.Service, error) {
	var err error
	service, _ := fAPI.getService(name)
	service.Annotations[opts.CustomAnnotation] = strconv.FormatBool(annotationEnabled)

	ctx, cancel := context.WithTimeout(parentCtx, 60*time.Second)
	defer cancel()
	service, err = fAPI.ClientSet.CoreV1().Services(fAPI.Namespace).Update(ctx, service, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}

	return service, nil
}

func TestRunServiceInformerCase1(t *testing.T) {
	/*
		- new NodePort type service added without required annotation
		- that service is updated with required annotation
	*/
	api := getFakeAPI()
	assert.NotNil(t, api)

	opts.Mu.Lock()
	opts.TemplateInputFile = "../../../resources/default.conf.tmpl"
	opts.Mu.Unlock()

	var clusters []*types.Cluster
	nginxConf := types.NewNginxConf(clusters)
	cluster := types.NewCluster("", make([]*types.Worker, 0))
	nginxConf.Clusters = append(nginxConf.Clusters, cluster)
	t.Logf(opts.CustomAnnotation)

	go func() {
		RunServiceInformer(cluster, api.ClientSet, logging.NewLogger(), opts, nginxConf)
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		time.Sleep(2 * time.Second)
		defer wg.Done()
		service, err := api.createService("nginx-a", true)
		assert.Nil(t, err)
		assert.NotNil(t, service)
		t.Logf("service created without required annotations")
	}()
	wg.Wait()

	wg.Add(1)
	go func() {
		time.Sleep(2 * time.Second)
		defer wg.Done()
		service, err := api.updateService("nginx-a", false)
		assert.NotNil(t, service)
		assert.Nil(t, err)
		t.Logf("service updated with required annotations")
	}()
	wg.Wait()

	wg.Add(1)
	go func() {
		time.Sleep(2 * time.Second)
		defer wg.Done()
		service, err := api.getService("nginx-a")
		assert.Nil(t, err)
		assert.NotNil(t, service)
		t.Logf("service fetched")
		assert.Equal(t, "false", service.Annotations[opts.CustomAnnotation])
	}()
	wg.Wait()
}
