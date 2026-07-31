package store

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

// PoolCollector exports pgxpool statistics as Prometheus gauges. A custom
// collector reads live values at scrape time — no background updating.
type PoolCollector struct {
	pool *pgxpool.Pool

	total    *prometheus.Desc
	acquired *prometheus.Desc
	idle     *prometheus.Desc
	max      *prometheus.Desc
	waitTime *prometheus.Desc
}

func NewPoolCollector(pool *pgxpool.Pool) *PoolCollector {
	return &PoolCollector{
		pool:     pool,
		total:    prometheus.NewDesc("mb_db_pool_total_conns", "Total connections in the pool.", nil, nil),
		acquired: prometheus.NewDesc("mb_db_pool_acquired_conns", "Connections currently in use.", nil, nil),
		idle:     prometheus.NewDesc("mb_db_pool_idle_conns", "Idle connections ready for use.", nil, nil),
		max:      prometheus.NewDesc("mb_db_pool_max_conns", "Configured pool ceiling.", nil, nil),
		waitTime: prometheus.NewDesc("mb_db_pool_acquire_wait_seconds_total", "Cumulative time callers waited for a connection (pool saturation signal).", nil, nil),
	}
}

func (c *PoolCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.total
	ch <- c.acquired
	ch <- c.idle
	ch <- c.max
	ch <- c.waitTime
}

func (c *PoolCollector) Collect(ch chan<- prometheus.Metric) {
	s := c.pool.Stat()
	ch <- prometheus.MustNewConstMetric(c.total, prometheus.GaugeValue, float64(s.TotalConns()))
	ch <- prometheus.MustNewConstMetric(c.acquired, prometheus.GaugeValue, float64(s.AcquiredConns()))
	ch <- prometheus.MustNewConstMetric(c.idle, prometheus.GaugeValue, float64(s.IdleConns()))
	ch <- prometheus.MustNewConstMetric(c.max, prometheus.GaugeValue, float64(s.MaxConns()))
	ch <- prometheus.MustNewConstMetric(c.waitTime, prometheus.CounterValue, s.AcquireDuration().Seconds())
}
