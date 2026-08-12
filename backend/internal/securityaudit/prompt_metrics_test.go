package securityaudit

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAtomicMetricsExposeCountsLatencyDistributionAndAsyncDelivery(t *testing.T) {
	metrics := NewAtomicMetrics()
	latencies := []time.Duration{10, 20, 30, 40, 100}
	kinds := []DecisionKind{DecisionAllow, DecisionFlag, DecisionBlock, DecisionUnavailable, DecisionInvalid}
	for index := range latencies {
		metrics.Observe(kinds[index], latencies[index]*time.Millisecond)
	}
	metrics.IncTimeout()
	metrics.IncFailover()
	metrics.IncBulkheadFull()
	metrics.IncRecordFailed()
	metrics.IncEnqueued()
	metrics.IncDropped()

	snapshot := metrics.Snapshot()
	require.Equal(t, int64(5), snapshot.Total)
	require.Equal(t, int64(5), snapshot.LatencyCount)
	require.Equal(t, int64(40), snapshot.LatencyAvgMS)
	require.Equal(t, int64(30), snapshot.LatencyP50MS)
	require.Equal(t, int64(40), snapshot.LatencyP95MS)
	require.Equal(t, int64(40), snapshot.LatencyP99MS)
	require.Equal(t, int64(100), snapshot.LatencyMaxMS)
	require.Equal(t, AuditMetricsSnapshot{Enqueued: 1, Dropped: 1}, metrics.AuditSnapshot())
}

func TestAtomicMetricsConcurrentObservationIsBoundedAndRaceSafe(t *testing.T) {
	metrics := NewAtomicMetrics()
	const observations = 4096
	var wg sync.WaitGroup
	for index := 0; index < observations; index++ {
		wg.Add(1)
		go func(value int) {
			defer wg.Done()
			metrics.Observe(DecisionAllow, time.Duration(value%250)*time.Millisecond)
		}(index)
	}
	wg.Wait()
	require.Equal(t, int64(observations), metrics.Snapshot().Total)
	metrics.latencyMu.RLock()
	require.LessOrEqual(t, len(metrics.latencies), latencySampleCapacity)
	metrics.latencyMu.RUnlock()
}

func TestAtomicMetricsKeepsEndpointObservationsIsolated(t *testing.T) {
	metrics := NewAtomicMetrics()
	metrics.ObserveEndpoint("guard-a", DecisionUnavailable, 80*time.Millisecond)
	metrics.IncEndpointTimeout("guard-a")
	metrics.IncEndpointFailover("guard-a")
	metrics.ObserveEndpoint("guard-b", DecisionAllow, 20*time.Millisecond)

	snapshots := metrics.EndpointSnapshots()
	require.Equal(t, int64(1), snapshots["guard-a"].Total)
	require.Equal(t, int64(1), snapshots["guard-a"].Unavailable)
	require.Equal(t, int64(1), snapshots["guard-a"].Timeouts)
	require.Equal(t, int64(1), snapshots["guard-a"].Failovers)
	require.Equal(t, int64(80), snapshots["guard-a"].LatencyP95MS)
	require.Equal(t, int64(1), snapshots["guard-b"].Allowed)
	require.Equal(t, int64(20), snapshots["guard-b"].LatencyP95MS)
	require.Equal(t, int64(0), metrics.Snapshot().Total)
}
