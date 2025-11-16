package observability

import (
	"net/http"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all Prometheus metrics for the daemon.
type Metrics struct {
	// Transfer metrics
	TransfersTotal        *prometheus.CounterVec
	TransfersActive       prometheus.Gauge
	TransferDuration      prometheus.Histogram
	BytesTransferredTotal *prometheus.CounterVec
	ChunksSentTotal       prometheus.Counter
	ChunksReceivedTotal   prometheus.Counter
	ChunksRetransmitted   *prometheus.CounterVec

	// Connection metrics
	QUICConnectionsTotal    *prometheus.CounterVec
	QUICConnectionsActive   prometheus.Gauge
	QUICConnectionDuration  prometheus.Histogram
	QUICStreamsActive       prometheus.Gauge
	QUICPacketLossRate      prometheus.Gauge
	FECEnabled              prometheus.Gauge
	FECReconstructionsTotal prometheus.Counter
	FECReconstructionFailuresTotal prometheus.Counter
	FECParityShardsSentTotal       prometheus.Counter

	// Crypto metrics
	CryptoOperationsTotal     *prometheus.CounterVec
	CryptoOperationDuration   prometheus.Histogram
	MerkleVerificationsTotal  *prometheus.CounterVec

	// Storage metrics
	BitmapPersistDuration   prometheus.Histogram
	DatabaseOperationsTotal *prometheus.CounterVec
	DiskSpaceUsedBytes      prometheus.Gauge

	// Active transfers counter (atomic for thread-safety)
	activeTransfers int64
}

// NewMetrics creates and registers all Prometheus metrics.
func NewMetrics() *Metrics {
	m := &Metrics{
		// Transfer metrics
		TransfersTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "quantarax_transfers_total",
				Help: "Total transfers initiated",
			},
			[]string{"status"},
		),

		TransfersActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "quantarax_transfers_active",
				Help: "Currently active transfers",
			},
		),

		TransferDuration: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "quantarax_transfer_duration_seconds",
				Help:    "Transfer completion time distribution",
				Buckets: []float64{1, 5, 10, 30, 60, 120, 300, 600, 1200, 1800},
			},
		),

		BytesTransferredTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "quantarax_bytes_transferred_total",
				Help: "Total bytes transferred",
			},
			[]string{"direction"},
		),

		ChunksSentTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "quantarax_chunks_sent_total",
				Help: "Total chunks sent",
			},
		),

		ChunksReceivedTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "quantarax_chunks_received_total",
				Help: "Total chunks received",
			},
		),

		ChunksRetransmitted: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "quantarax_chunks_retransmitted_total",
				Help: "Chunks requiring retransmission",
			},
			[]string{"reason"},
		),

		// Connection metrics
		QUICConnectionsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "quantarax_quic_connections_total",
				Help: "QUIC connection attempts",
			},
			[]string{"result"},
		),

		QUICConnectionsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "quantarax_quic_connections_active",
				Help: "Active QUIC connections",
			},
		),

		QUICConnectionDuration: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "quantarax_quic_connection_duration_seconds",
				Help:    "QUIC connection lifetime",
				Buckets: []float64{1, 5, 10, 30, 60, 120, 300},
			},
		),

		QUICStreamsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "quantarax_quic_streams_active",
				Help: "Active QUIC streams",
			},
		),

		QUICPacketLossRate: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "quantarax_quic_packet_loss_rate",
				Help: "Observed packet loss rate (0.0–1.0)",
			},
		),

		// FEC metrics
		FECEnabled: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "quantarax_fec_enabled",
				Help: "FEC currently enabled (0/1)",
			},
		),

		FECReconstructionsTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "quantarax_fec_reconstructions_total",
				Help: "Chunks reconstructed via FEC",
			},
		),

		FECReconstructionFailuresTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "quantarax_fec_reconstruction_failures_total",
				Help: "Failed FEC reconstructions",
			},
		),

		FECParityShardsSentTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "quantarax_fec_parity_shards_sent_total",
				Help: "Parity shards transmitted",
			},
		),

		// Crypto metrics
		CryptoOperationsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "quantarax_crypto_operations_total",
				Help: "Cryptographic operations performed",
			},
			[]string{"operation"},
		),

		CryptoOperationDuration: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "quantarax_crypto_operation_duration_seconds",
				Help:    "Crypto operation latency",
				Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0},
			},
		),

		MerkleVerificationsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "quantarax_merkle_verifications_total",
				Help: "Merkle root verifications",
			},
			[]string{"result"},
		),

		// Storage metrics
		BitmapPersistDuration: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "quantarax_bitmap_persist_duration_seconds",
				Help:    "Bitmap persistence latency",
				Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1.0, 2.0},
			},
		),

		DatabaseOperationsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "quantarax_database_operations_total",
				Help: "Database operation count",
			},
			[]string{"operation", "result"},
		),

		DiskSpaceUsedBytes: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "quantarax_disk_space_used_bytes",
				Help: "Disk space used by received files",
			},
		),
	}

	return m
}

// RecordTransferStart increments active transfer counters.
func (m *Metrics) RecordTransferStart() {
	atomic.AddInt64(&m.activeTransfers, 1)
	m.TransfersActive.Set(float64(atomic.LoadInt64(&m.activeTransfers)))
}

// RecordTransferComplete records transfer completion metrics.
func (m *Metrics) RecordTransferComplete(success bool, durationSeconds float64) {
	atomic.AddInt64(&m.activeTransfers, -1)
	m.TransfersActive.Set(float64(atomic.LoadInt64(&m.activeTransfers)))

	status := "success"
	if !success {
		status = "failure"
	}

	m.TransfersTotal.WithLabelValues(status).Inc()
	m.TransferDuration.Observe(durationSeconds)
}

// RecordChunkSent updates metrics for a sent chunk.
func (m *Metrics) RecordChunkSent(bytes int) {
	m.ChunksSentTotal.Inc()
	m.BytesTransferredTotal.WithLabelValues("sent").Add(float64(bytes))
}

// RecordChunkReceived updates metrics for a received chunk.
func (m *Metrics) RecordChunkReceived(bytes int) {
	m.ChunksReceivedTotal.Inc()
	m.BytesTransferredTotal.WithLabelValues("received").Add(float64(bytes))
}

// RecordChunkRetransmit increments retransmit counters.
func (m *Metrics) RecordChunkRetransmit(reason string) {
	m.ChunksRetransmitted.WithLabelValues(reason).Inc()
}

// RecordQUICConnection logs QUIC connection attempts.
func (m *Metrics) RecordQUICConnection(success bool) {
	result := "success"
	if !success {
		result = "failure"
	}
	m.QUICConnectionsTotal.WithLabelValues(result).Inc()

	if success {
		m.QUICConnectionsActive.Inc()
	}
}

// RecordQUICConnectionClose updates metrics for closed QUIC connections.
func (m *Metrics) RecordQUICConnectionClose(durationSeconds float64) {
	m.QUICConnectionsActive.Dec()
	m.QUICConnectionDuration.Observe(durationSeconds)
}

// RecordCryptoOperation records cryptographic operation duration.
func (m *Metrics) RecordCryptoOperation(operation string, durationSeconds float64) {
	m.CryptoOperationsTotal.WithLabelValues(operation).Inc()
	m.CryptoOperationDuration.Observe(durationSeconds)
}

// RecordMerkleVerification increments Merkle verification counters.
func (m *Metrics) RecordMerkleVerification(success bool) {
	result := "success"
	if !success {
		result = "failure"
	}
	m.MerkleVerificationsTotal.WithLabelValues(result).Inc()
}

// RecordFECReconstruction updates FEC reconstruction counters.
func (m *Metrics) RecordFECReconstruction(success bool) {
	if success {
		m.FECReconstructionsTotal.Inc()
	} else {
		m.FECReconstructionFailuresTotal.Inc()
	}
}

// SetFECEnabled sets the FEC enabled flag.
func (m *Metrics) SetFECEnabled(enabled bool) {
	if enabled {
		m.FECEnabled.Set(1)
	} else {
		m.FECEnabled.Set(0)
	}
}

// Handler exposes the Prometheus metrics endpoint.
func (m *Metrics) Handler() http.Handler {
	return promhttp.Handler()
}
