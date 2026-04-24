package postgres

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_StatsCollector_EXPORTS_ALL_METRICS(t *testing.T) {
	db := openTestDB(t)
	sqlDB, err := db.DB()
	require.NoError(t, err,
		"expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	reg := prometheus.NewPedanticRegistry()
	collector := NewStatsCollector(sqlDB, "testdb", "localhost")
	reg.MustRegister(collector)

	families, err := reg.Gather()
	require.NoError(t, err,
		"expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	expected := map[string]bool{
		"db_pool_max_open_connections":        false,
		"db_pool_open_connections":            false,
		"db_pool_in_use_connections":          false,
		"db_pool_idle_connections":            false,
		"db_pool_wait_count_total":            false,
		"db_pool_wait_duration_seconds_total": false,
		"db_pool_max_idle_closed_total":       false,
		"db_pool_max_idle_time_closed_total":  false,
		"db_pool_max_lifetime_closed_total":   false,
	}

	for _, fam := range families {
		if _, ok := expected[fam.GetName()]; ok {
			expected[fam.GetName()] = true

			for _, m := range fam.GetMetric() {
				labels := m.GetLabel()
				assert.Len(t, labels, 2,
					"metric %s should have db and host labels",
					fam.GetName())
			}
		}
	}

	for name, found := range expected {
		assert.True(t, found, "metric %s was not exported", name)
	}
}

func Test_StatsCollector_DESCRIBE_RETURNS_NINE_DESCRIPTORS(t *testing.T) {
	db := openTestDB(t)
	sqlDB, err := db.DB()
	require.NoError(t, err,
		"expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	collector := NewStatsCollector(sqlDB, "testdb", "replica.local")
	ch := make(chan *prometheus.Desc, 16)
	collector.Describe(ch)
	close(ch)

	count := 0
	for range ch {
		count++
	}
	assert.Equal(t, 9, count)
}

func Test_StatsCollector_DISTINCT_HOSTS_COEXIST(t *testing.T) {
	writerDB := openTestDB(t)
	readerDB := openTestDB(t)

	writerSQL, err := writerDB.DB()
	require.NoError(t, err,
		"expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	readerSQL, err := readerDB.DB()
	require.NoError(t, err,
		"expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	reg := prometheus.NewPedanticRegistry()
	reg.MustRegister(NewStatsCollector(writerSQL, "mydb", "primary.local"))
	reg.MustRegister(NewStatsCollector(readerSQL, "mydb", "replica.local"))

	count := testutil.CollectAndCount(
		NewStatsCollector(writerSQL, "mydb", "primary.local"),
	)
	assert.Equal(t, 9, count, "each collector should emit 9 metrics")
}

func Test_RegisterPoolMetrics_DUPLICATE_RETURNS_NIL(t *testing.T) {
	db := openTestDB(t)
	sqlDB, err := db.DB()
	require.NoError(t, err,
		"expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	reg := prometheus.NewRegistry()
	saved := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = reg
	t.Cleanup(func() { prometheus.DefaultRegisterer = saved })

	err = RegisterPoolMetrics(sqlDB, "dupdb", "localhost")
	require.NoError(t, err,
		"expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	err = RegisterPoolMetrics(sqlDB, "dupdb", "localhost")
	assert.NoError(t, err,
		"expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
}
