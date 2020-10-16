package prom

import (
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const statsSubsystem = "stats"

var (
	metrics bool

	receipts     prometheus.Counter
	transactions prometheus.Counter
	blocks       prometheus.Counter

	tPayloadDecode           prometheus.Histogram // payload decoding time: 58.173µs
	tFreePostgres            prometheus.Histogram // time spent waiting for free postgres tx: 94.511µs
	tPostgresCommit          prometheus.Histogram // postgres transaction commit duration: 1.509643ms
	tHeaderProcessing        prometheus.Histogram // header processing time: 1.057721ms
	tUncleProcessing         prometheus.Histogram // uncle processing time: 140ns
	tTxAndRecProcessing      prometheus.Histogram // tx and receipt processing time: 1.749µs
	tStateAndStoreProcessing prometheus.Histogram // state and storage processing time: 4.737994ms
)

// Init module initialization
func Init() {
	metrics = true

	blocks = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "blocks",
		Help:      "The total number of processed blocks",
	})
	transactions = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "transactions",
		Help:      "The total number of processed transactions",
	})
	receipts = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "receipts",
		Help:      "The total number of processed receipts",
	})

	tPayloadDecode = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: statsSubsystem,
		Name:      "t_payload_decode",
		Help:      "Payload decoding time",
	})
	tFreePostgres = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: statsSubsystem,
		Name:      "t_free_postgres",
		Help:      "Time spent waiting for free postgres tx",
	})
	tPostgresCommit = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: statsSubsystem,
		Name:      "t_postgres_commit",
		Help:      "Postgres transaction commit duration",
	})
	tHeaderProcessing = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: statsSubsystem,
		Name:      "t_header_processing",
		Help:      "Header processing time",
	})
	tUncleProcessing = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: statsSubsystem,
		Name:      "t_uncle_processing",
		Help:      "Uncle processing time",
	})
	tTxAndRecProcessing = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: statsSubsystem,
		Name:      "t_tx_receipt_processing",
		Help:      "Tx and receipt processing time",
	})
	tStateAndStoreProcessing = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: statsSubsystem,
		Name:      "t_state_store_processing",
		Help:      "State and storage processing time",
	})
}

// RegisterDBCollector create metric colletor for given connection
func RegisterDBCollector(name string, db *sqlx.DB) {
	if metrics {
		prometheus.Register(NewDBStatsCollector(name, db))
	}
}

// BlockInc block counter increment
func BlockInc() {
	if metrics {
		blocks.Inc()
	}
}

// TransactionInc transaction counter increment
func TransactionInc() {
	if metrics {
		transactions.Inc()
	}
}

// ReceiptInc receipt counter increment
func ReceiptInc() {
	if metrics {
		receipts.Inc()
	}
}

// SetTimeMetric time metric observation
func SetTimeMetric(name string, t time.Duration) {
	if !metrics {
		return
	}
	tAsF64 := t.Seconds()
	switch name {
	case "t_payload_decode":
		tPayloadDecode.Observe(tAsF64)
	case "t_free_postgres":
		tFreePostgres.Observe(tAsF64)
	case "t_postgres_commit":
		tPostgresCommit.Observe(tAsF64)
	case "t_header_processing":
		tHeaderProcessing.Observe(tAsF64)
	case "t_uncle_processing":
		tUncleProcessing.Observe(tAsF64)
	case "t_tx_receipt_processing":
		tTxAndRecProcessing.Observe(tAsF64)
	case "t_state_store_processing":
		tStateAndStoreProcessing.Observe(tAsF64)
	}
}
