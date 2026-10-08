package metrics

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

// PoolStatter is satisfied by *pgxpool.Pool.
type PoolStatter interface {
	Stat() *pgxpool.Stat
}

// RegisterPool exposes pgxpool statistics. The collector reads Stat() at
// scrape time, so there is no background goroutine and no stale value.
//
// The two numbers that matter for saturation are db_pool_empty_acquires_total
// (how often a caller had to WAIT because every connection was busy) and
// db_pool_acquire_duration_seconds_total (how long callers waited in total).
func (m *Metrics) RegisterPool(pool PoolStatter) {
	m.Registry.MustRegister(&poolCollector{pool: pool})
}

type poolCollector struct {
	pool PoolStatter
}

var (
	descConns = prometheus.NewDesc("db_pool_connections",
		"Pool connections by state (acquired = in use, idle, constructing).", []string{"state"}, nil)
	descMax = prometheus.NewDesc("db_pool_max_connections",
		"Configured maximum pool size.", nil, nil)
	descTotal = prometheus.NewDesc("db_pool_total_connections",
		"Connections currently in the pool (acquired + idle + constructing).", nil, nil)
	descAcquires = prometheus.NewDesc("db_pool_acquires_total",
		"Successful connection acquires.", nil, nil)
	descWaited = prometheus.NewDesc("db_pool_empty_acquires_total",
		"Acquires that had to wait because no idle connection was available.", nil, nil)
	descWaitSeconds = prometheus.NewDesc("db_pool_acquire_duration_seconds_total",
		"Total time callers spent acquiring a connection.", nil, nil)
	descCanceled = prometheus.NewDesc("db_pool_canceled_acquires_total",
		"Acquires abandoned because the caller's context ended first.", nil, nil)
	descNew = prometheus.NewDesc("db_pool_new_connections_total",
		"Connections opened over the pool's lifetime.", nil, nil)
)

func (c *poolCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, d := range []*prometheus.Desc{descConns, descMax, descTotal, descAcquires,
		descWaited, descWaitSeconds, descCanceled, descNew} {
		ch <- d
	}
}

func (c *poolCollector) Collect(ch chan<- prometheus.Metric) {
	s := c.pool.Stat()
	g := prometheus.GaugeValue
	ct := prometheus.CounterValue

	ch <- prometheus.MustNewConstMetric(descConns, g, float64(s.AcquiredConns()), "acquired")
	ch <- prometheus.MustNewConstMetric(descConns, g, float64(s.IdleConns()), "idle")
	ch <- prometheus.MustNewConstMetric(descConns, g, float64(s.ConstructingConns()), "constructing")
	ch <- prometheus.MustNewConstMetric(descMax, g, float64(s.MaxConns()))
	ch <- prometheus.MustNewConstMetric(descTotal, g, float64(s.TotalConns()))
	ch <- prometheus.MustNewConstMetric(descAcquires, ct, float64(s.AcquireCount()))
	ch <- prometheus.MustNewConstMetric(descWaited, ct, float64(s.EmptyAcquireCount()))
	ch <- prometheus.MustNewConstMetric(descWaitSeconds, ct, s.AcquireDuration().Seconds())
	ch <- prometheus.MustNewConstMetric(descCanceled, ct, float64(s.CanceledAcquireCount()))
	ch <- prometheus.MustNewConstMetric(descNew, ct, float64(s.NewConnsCount()))
}
