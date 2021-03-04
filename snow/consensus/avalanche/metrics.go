// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package avalanche

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/logging"
	"github.com/ava-labs/avalanchego/utils/timer"
	"github.com/ava-labs/avalanchego/utils/wrappers"
)

type metrics struct {
	numProcessing            prometheus.Gauge
	latAccepted, latRejected prometheus.Histogram
	log                      logging.Logger

	clock timer.Clock

	processingTxs *ProcessingTxs
}

// Initialize implements the Engine interface
func (m *metrics) Initialize(log logging.Logger, namespace string, registerer prometheus.Registerer) error {
	m.processingTxs = NewProcessingTxs()
	m.log = log

	m.numProcessing = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "vtx_processing",
		Help:      "Number of currently processing vertices",
	})
	m.latAccepted = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "vtx_accepted",
		Help:      "Latency of accepting from the time the vertex was issued in milliseconds",
		Buckets:   timer.MillisecondsBuckets,
	})
	m.latRejected = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "vtx_rejected",
		Help:      "Latency of rejecting from the time the vertex was issued in milliseconds",
		Buckets:   timer.MillisecondsBuckets,
	})

	errs := wrappers.Errs{}
	errs.Add(
		registerer.Register(m.numProcessing),
		registerer.Register(m.latAccepted),
		registerer.Register(m.latRejected),
	)
	return errs.Err
}

func (m *metrics) Issued(id ids.ID) {
	m.processingTxs.PutTx(id, m.clock.Time())
	m.numProcessing.Inc()
}

func (m *metrics) Accepted(id ids.ID) {
	start, ok := m.processingTxs.GetTx(id)
	if !ok {
		m.log.Debug("unable to measure Accepted transaction %v", id.String())
		return
	}

	end := m.clock.Time()

	m.processingTxs.Evict(id)

	m.latAccepted.Observe(float64(end.Sub(start.Time).Milliseconds()))
	m.numProcessing.Dec()
}

func (m *metrics) Rejected(id ids.ID) {
	start, ok := m.processingTxs.GetTx(id)
	if !ok {
		m.log.Debug("unable to measure Rejected transaction %v", id.String())
		return
	}
	end := m.clock.Time()

	m.processingTxs.Evict(id)

	m.latRejected.Observe(float64(end.Sub(start.Time).Milliseconds()))
	m.numProcessing.Dec()
}
