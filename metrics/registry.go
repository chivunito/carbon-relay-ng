package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

var Registerer prometheus.Registerer
var Gatherer prometheus.Gatherer

type SafeRegistry struct {
	*prometheus.Registry
}

func NewSafeRegistry() *SafeRegistry {
	return &SafeRegistry{
		prometheus.NewRegistry(),
	}
}

func (r *SafeRegistry) MustRegister(collectors ...prometheus.Collector) {
	for _, cltor := range collectors {
		r.Unregister(cltor)
		if err := r.Register(cltor); err != nil {
			zap.S().Warnf("failed to register a collector: %s", err)
		}
	}
}

func init() {
	sr := NewSafeRegistry()
	prometheus.DefaultRegisterer = sr
	prometheus.DefaultGatherer = sr
	sr.MustRegister(prometheus.NewGoCollector())
	registerInputMetrics(sr)
}
