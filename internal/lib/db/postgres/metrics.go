package postgres

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

// Exports sql.DBStats as Prometheus metrics, providing live visibility
// into connection pool utilisation on each scrape without a polling
// goroutine.
type StatsCollector struct {
	db *sql.DB

	maxOpen           *prometheus.Desc
	open              *prometheus.Desc
	inUse             *prometheus.Desc
	idle              *prometheus.Desc
	waitCount         *prometheus.Desc
	waitDuration      *prometheus.Desc
	maxIdleClosed     *prometheus.Desc
	maxIdleTimeClosed *prometheus.Desc
	maxLifetimeClosed *prometheus.Desc
}

// Creates a collector that reads sql.DBStats on every Prometheus scrape.
// The dbName and host labels distinguish pools when a process holds more
// than one connection (e.g. writer vs reader replicas that point to
// different hosts).
func NewStatsCollector(
	db *sql.DB,
	dbName, host string,
) *StatsCollector {
	labels := prometheus.Labels{"db": dbName, "host": host}

	return &StatsCollector{
		db: db,
		maxOpen: prometheus.NewDesc(
			"db_pool_max_open_connections",
			"Maximum number of open connections to the database",
			nil, labels,
		),
		open: prometheus.NewDesc(
			"db_pool_open_connections",
			"Current number of open connections (in-use + idle)",
			nil, labels,
		),
		inUse: prometheus.NewDesc(
			"db_pool_in_use_connections",
			"Connections currently in use",
			nil, labels,
		),
		idle: prometheus.NewDesc(
			"db_pool_idle_connections",
			"Connections currently idle",
			nil, labels,
		),
		waitCount: prometheus.NewDesc(
			"db_pool_wait_count_total",
			"Total number of connections waited for",
			nil, labels,
		),
		waitDuration: prometheus.NewDesc(
			"db_pool_wait_duration_seconds_total",
			"Total time blocked waiting for a new connection",
			nil, labels,
		),
		maxIdleClosed: prometheus.NewDesc(
			"db_pool_max_idle_closed_total",
			"Connections closed because max idle count was reached",
			nil, labels,
		),
		maxIdleTimeClosed: prometheus.NewDesc(
			"db_pool_max_idle_time_closed_total",
			"Connections closed because max idle time was reached",
			nil, labels,
		),
		maxLifetimeClosed: prometheus.NewDesc(
			"db_pool_max_lifetime_closed_total",
			"Connections closed because max lifetime was reached",
			nil, labels,
		),
	}
}

func (c *StatsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.maxOpen
	ch <- c.open
	ch <- c.inUse
	ch <- c.idle
	ch <- c.waitCount
	ch <- c.waitDuration
	ch <- c.maxIdleClosed
	ch <- c.maxIdleTimeClosed
	ch <- c.maxLifetimeClosed
}

func (c *StatsCollector) Collect(ch chan<- prometheus.Metric) {
	s := c.db.Stats()

	ch <- prometheus.MustNewConstMetric(
		c.maxOpen, prometheus.GaugeValue,
		float64(s.MaxOpenConnections),
	)
	ch <- prometheus.MustNewConstMetric(
		c.open, prometheus.GaugeValue,
		float64(s.OpenConnections),
	)
	ch <- prometheus.MustNewConstMetric(
		c.inUse, prometheus.GaugeValue,
		float64(s.InUse),
	)
	ch <- prometheus.MustNewConstMetric(
		c.idle, prometheus.GaugeValue,
		float64(s.Idle),
	)
	ch <- prometheus.MustNewConstMetric(
		c.waitCount, prometheus.CounterValue,
		float64(s.WaitCount),
	)
	ch <- prometheus.MustNewConstMetric(
		c.waitDuration, prometheus.CounterValue,
		s.WaitDuration.Seconds(),
	)
	ch <- prometheus.MustNewConstMetric(
		c.maxIdleClosed, prometheus.CounterValue,
		float64(s.MaxIdleClosed),
	)
	ch <- prometheus.MustNewConstMetric(
		c.maxIdleTimeClosed, prometheus.CounterValue,
		float64(s.MaxIdleTimeClosed),
	)
	ch <- prometheus.MustNewConstMetric(
		c.maxLifetimeClosed, prometheus.CounterValue,
		float64(s.MaxLifetimeClosed),
	)
}

// Registers a StatsCollector for the given sql.DB on the default Prometheus
// registry. Duplicate registrations (same dbName/host pair) return nil.
// Any other registration error is returned to the caller.
func RegisterPoolMetrics(db *sql.DB, dbName, host string) error {
	c := NewStatsCollector(db, dbName, host)

	if err := prometheus.Register(c); err != nil {
		var already prometheus.AlreadyRegisteredError
		if errors.As(err, &already) {
			return nil
		}
		return fmt.Errorf(
			"failed to register db pool metrics (db=%s, host=%s): %w",
			dbName, host, err,
		)
	}

	return nil
}
