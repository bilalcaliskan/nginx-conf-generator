package metrics

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"nginx-conf-generator/internal/options"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRunMetricsServer(t *testing.T) {
	opts := options.GetNginxConfGeneratorOptions()
	var (
		conn net.Conn
		err  error
	)

	defer func() {
		err := conn.Close()
		assert.Nil(t, err)
	}()

	go func() {
		err := RunMetricsServer()
		assert.Nil(t, err)
	}()

	for {
		time.Sleep(1 * time.Second)
		conn, _ = net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", opts.MetricsPort), 10*time.Second)
		if conn != nil {
			break
		}
	}

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d%s", opts.MetricsPort, opts.MetricsEndpoint))
	assert.Nil(t, err)

	body, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)

	assert.Contains(t, string(body), ProcessedNodePortCounterName)
	assert.Contains(t, string(body), TargetNodePortCounterName)
}
