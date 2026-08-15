package metrics

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	Hits *prometheus.CounterVec
}

func New() *Metrics {
	metrics := &Metrics{
		Hits: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "hits",
			Help: "The total number of requests made",
		}, []string{"status", "path"}),
	}
	return metrics
}
