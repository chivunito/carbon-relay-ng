package route

import (
	"context"
	"testing"
	"time"

	"github.com/graphite-ng/carbon-relay-ng/encoding"
	"github.com/graphite-ng/carbon-relay-ng/storage"
	"github.com/stretchr/testify/assert"
)

func testBloomFilterConfig() BloomFilterConfig {
	const (
		shardingFactor = 3
		filterSize     = 1000000
		faultTolerance = 0.0000001
		clearInterval  = time.Duration(10 * time.Millisecond)
		saveInterval   = time.Duration(10 * time.Millisecond)
		cache          = "" // don't test caching
	)
	var clearWait time.Duration
	bfc, _ := NewBloomFilterConfig(
		filterSize,
		faultTolerance,
		shardingFactor,
		cache,
		clearInterval,
		clearWait,
		saveInterval,
	)
	return bfc
}

func testBgMetadata(t *testing.T) *BgMetadata {
	const (
		key      = "test_route"
		prefix   = ""
		sub      = ""
		regex    = ""
		notRegex = ""
		sch      = "../examples/storage-schemas.conf"
		agg      = "../examples/storage-aggregation.conf"
	)
	bfc := testBloomFilterConfig()
	m, _ := NewBgMetadataRoute(key, prefix, sub, regex, notRegex, agg, sch, bfc, "testing", nil, "")
	m.ctx, m.cancel = context.WithCancel(context.Background())
	return m
}

func TestMetricFiltering(t *testing.T) {
	m := testBgMetadata(t)
	dp := make([]encoding.Datapoint, m.bfCfg.ShardingFactor)
	dp[0] = encoding.Datapoint{Name: "metric.name.aaaa"}      // 0
	dp[1] = encoding.Datapoint{Name: "metric.name.aaaaaaaaa"} // 1
	dp[2] = encoding.Datapoint{Name: "metric.name.bbbb"}      // 2

	// check if right filters are filled
	for i := 0; i < len(dp); i++ {
		m.Dispatch(dp[i])
		assert.True(t, m.shards[i].filter.TestString(dp[i].Name))
	}
	m.Dispatch(dp[0])

	// check if added metrics are cleared from the filters
	go m.clearBloomFilter()
	time.Sleep(30 * time.Millisecond)
	m.Shutdown()
	for i := 0; i < len(dp); i++ {
		assert.False(t, m.shards[i].filter.TestString(dp[i].Name))
	}
}

func TestMetricUpdated(t *testing.T) {
	m := testBgMetadata(t)
	m.Dispatch(encoding.Datapoint{Name: "metric.name.aaaa"})
	m.Shutdown()
	conn := m.storage.(*storage.BgMetadataTestingStorageConnector)
	assert.Contains(t, conn.UpdatedMetrics, "metric.name.aaaa")
}

func TestDirectoriesUpdated(t *testing.T) {
	m := testBgMetadata(t)
	m.Dispatch(encoding.Datapoint{Name: "metric.name.aaaa"})
	m.Shutdown()
	conn := m.storage.(*storage.BgMetadataTestingStorageConnector)

	assert.Contains(t, conn.UpdatedDirectories, "metric.name")
	assert.Contains(t, conn.UpdatedDirectories, "metric")
	assert.NotContains(t, conn.UpdatedDirectories, "metric.name.aaaa")
}

func TestClearWaitDefaultValue(t *testing.T) {
	bfc := testBloomFilterConfig()
	assert.Equal(t, bfc.ClearWait, bfc.ClearInterval/time.Duration(bfc.ShardingFactor))
}
