package informers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetClientSet(t *testing.T) {
	restConfig, err := GetConfig("../../../mock/kubeconfig")
	assert.Nil(t, err)
	assert.NotNil(t, restConfig)

	clientSet, err := GetClientSet(restConfig)
	assert.Nil(t, err)
	assert.NotNil(t, clientSet)

	restConfig, err = GetConfig("../../../mock/broken_kubeconfig")
	assert.NotNil(t, err)
	assert.Nil(t, restConfig)
}
