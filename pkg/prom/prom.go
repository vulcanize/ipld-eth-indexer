package prom

import (
	"github.com/jmoiron/sqlx"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	metrics      bool
	receipts     prometheus.Counter
	transactions prometheus.Counter
	blocks       prometheus.Counter
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
