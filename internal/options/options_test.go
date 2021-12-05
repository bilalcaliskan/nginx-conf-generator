package options

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestGetNginxConfGeneratorOptions function tests if GetNginxConfGeneratorOptions function running properly
func TestGetNginxConfGeneratorOptions(t *testing.T) {
	t.Log("fetching default options.NginxConfGeneratorOptions")
	opts := GetNginxConfGeneratorOptions()
	assert.NotNil(t, opts)
	t.Logf("fetched default options.NginxConfGeneratorOptions, %v\n", opts)
}
