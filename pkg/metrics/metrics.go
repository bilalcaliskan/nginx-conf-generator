package metrics

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"net/http"
	"nginx-conf-generator/pkg/logging"
	"nginx-conf-generator/pkg/options"
	"time"
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
		Name: "processed_nodeport_counter",
		Help: "Counts processed nodeport type services",
	})
	TargetNodeCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "target_node_counter",
		Help: "Counts target nodes on the managed clusters",
	})
}

// RunMetricsServer spins up a router to provide prometheus metrics
func RunMetricsServer() {
	defer func() {
		err := logger.Sync()
		if err != nil {
			panic(err)
		}
	}()

	router := mux.NewRouter()
	metricServer := &http.Server{
		Handler:      router,
		Addr:         fmt.Sprintf(":%d", opts.MetricsPort),
		WriteTimeout: time.Duration(int32(opts.WriteTimeoutSeconds)) * time.Second,
		ReadTimeout:  time.Duration(int32(opts.ReadTimeoutSeconds)) * time.Second,
	}
	router.Handle(opts.MetricsEndpoint, promhttp.Handler())
	prometheus.MustRegister(ProcessedNodePortCounter)
	prometheus.MustRegister(TargetNodeCounter)
	logger.Info("metric server is up and running", zap.Int("port", opts.MetricsPort))
	panic(metricServer.ListenAndServe())
}
