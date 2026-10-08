package metrics

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestPoolCollector_ReportsStatsWithoutADatabase(t *testing.T) {
	// pgxpool connects lazily, so a pool pointed at a closed port still has
	// valid (all-zero) stats — enough to prove the collector wiring.
	cfg, err := pgxpool.ParseConfig("postgres://u:p@127.0.0.1:1/db?pool_max_conns=7")
	if err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	m := New(nil)
	m.RegisterPool(pool)

	if got := testutil.ToFloat64(gatherOne(t, m, "db_pool_max_connections")); got != 7 {
		t.Errorf("db_pool_max_connections = %v, want 7", got)
	}
	families, err := m.Registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, f := range families {
		seen[f.GetName()] = true
	}
	for _, name := range []string{
		"db_pool_connections", "db_pool_max_connections", "db_pool_total_connections",
		"db_pool_acquires_total", "db_pool_empty_acquires_total",
		"db_pool_acquire_duration_seconds_total", "db_pool_canceled_acquires_total",
		"db_pool_new_connections_total",
	} {
		if !seen[name] {
			t.Errorf("pool metric %q not exposed", name)
		}
	}
}

// gatherOne returns a single-metric collector for the named family so
// testutil.ToFloat64 can read its value.
func gatherOne(t *testing.T, m *Metrics, name string) prometheus.Collector {
	t.Helper()
	families, err := m.Registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range families {
		if f.GetName() == name && len(f.GetMetric()) == 1 {
			return constCollector{desc: prometheus.NewDesc(name, "", nil, nil), value: f.GetMetric()[0].GetGauge().GetValue()}
		}
	}
	t.Fatalf("metric %q not found (or not a single gauge)", name)
	return nil
}

type constCollector struct {
	desc  *prometheus.Desc
	value float64
}

func (c constCollector) Describe(ch chan<- *prometheus.Desc) { ch <- c.desc }
func (c constCollector) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, c.value)
}
