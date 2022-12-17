package metrics

import (
	"fmt"
	"net/http"
	"time"

	"github.com/bilalcaliskan/nginx-conf-generator/internal/logging"
	"github.com/bilalcaliskan/nginx-conf-generator/internal/options"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

const (
	ProcessedNodePortCounterName = "processed_nodeport_counter"
	TargetNodePortCounterName    = "target_node_counter"
)

var (
	logger *zap.Logger
	opts   *options.NginxConfGeneratorOptions
	// ProcessedNodePortCounter keeps track of processed nodePort type service counter
	ProcessedNodePortCounter prometheus.Counter
	// TargetNodeCounter keeps track of the target nodes on the managed clusters
	TargetNodeCounter prometheus.Counter
)

func init() {
	logger = logging.GetLogger()
	opts = options.GetNginxConfGeneratorOptions()
	ProcessedNodePortCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: ProcessedNodePortCounterName,
		Help: "Counts processed nodeport type services",
	})
	TargetNodeCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: TargetNodePortCounterName,
		Help: "Counts target nodes on the managed clusters",
	})
}

// RunMetricsServer spins up a router to provide prometheus metrics
func RunMetricsServer() error {
	router := mux.NewRouter()
	metricServer := &http.Server{
		Handler:      router,
		Addr:         fmt.Sprintf(":%d", opts.MetricsPort),
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}
	router.Handle(opts.MetricsEndpoint, promhttp.Handler())
	prometheus.MustRegister(ProcessedNodePortCounter)
	prometheus.MustRegister(TargetNodeCounter)
	logger.Info("metric server is up and running", zap.Int("port", opts.MetricsPort))
	return metricServer.ListenAndServe()
}
