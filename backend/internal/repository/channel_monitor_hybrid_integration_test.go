//go:build integration

package repository

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/ent/channelmonitor"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestChannelMonitorTrafficBucket_ConcurrentUpsertAndAvailability(t *testing.T) {
	ctx := context.Background()
	user, err := integrationEntClient.User.Create().
		SetEmail(fmt.Sprintf("monitor-hybrid-%d@example.com", time.Now().UnixNano())).
		SetPasswordHash("test-password-hash").
		Save(ctx)
	require.NoError(t, err)

	monitor, err := integrationEntClient.ChannelMonitor.Create().
		SetName("hybrid integration monitor").
		SetProvider(channelmonitor.ProviderOpenai).
		SetEndpoint("https://example.invalid").
		SetAPIKeyEncrypted("encrypted-test-key").
		SetPrimaryModel("gpt-5.5").
		SetMode(channelmonitor.ModeHybrid).
		SetIntervalSeconds(3600).
		SetCreatedBy(user.ID).
		Save(ctx)
	require.NoError(t, err)

	repo := NewChannelMonitorRepository(integrationEntClient, integrationDB)
	bucketStart := time.Now().UTC().Add(-2 * time.Minute).Truncate(time.Minute)
	const writers = 16
	errCh := make(chan error, writers)
	var wg sync.WaitGroup
	for i := 0; i < writers; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			status := service.MonitorStatusOperational
			successes, failures := 1, 0
			if i%2 == 1 {
				status = service.MonitorStatusDegraded
				successes, failures = 0, 1
			}
			errCh <- repo.UpsertTrafficBucket(ctx, &service.ChannelMonitorHistoryRow{
				MonitorID: monitor.ID, Model: "gpt-concurrent", Status: status,
				Source: service.MonitorSourceRealTraffic, BucketStart: &bucketStart,
				SampleCount: 1, SuccessCount: successes, FailureCount: failures,
				CheckedAt: bucketStart.Add(time.Minute),
			})
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		require.NoError(t, err)
	}

	var count int
	err = integrationDB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM channel_monitor_histories
		WHERE monitor_id = $1 AND model = 'gpt-concurrent'
		  AND source = 'real_traffic' AND bucket_start = $2
	`, monitor.ID, bucketStart).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count, "concurrent writes must remain idempotent per minute bucket")

	availabilityModel := "gpt-availability"
	for i, status := range []string{service.MonitorStatusOperational, service.MonitorStatusError} {
		bucket := bucketStart.Add(time.Duration(i) * time.Minute)
		successes, failures := 1, 0
		if status == service.MonitorStatusError {
			successes, failures = 0, 1
		}
		row := &service.ChannelMonitorHistoryRow{
			MonitorID: monitor.ID, Model: availabilityModel, Status: status,
			Source: service.MonitorSourceRealTraffic, BucketStart: &bucket,
			SampleCount: 10, SuccessCount: successes, FailureCount: failures,
			CheckedAt: bucket.Add(time.Minute),
		}
		require.NoError(t, repo.UpsertTrafficBucket(ctx, row))
		require.NoError(t, repo.UpsertTrafficBucket(ctx, row))
	}

	availability, err := repo.ComputeAvailability(ctx, monitor.ID, 7)
	require.NoError(t, err)
	var found *service.ChannelMonitorAvailability
	for _, item := range availability {
		if item.Model == availabilityModel {
			found = item
			break
		}
	}
	require.NotNil(t, found)
	require.Equal(t, int64(2), found.TotalChecks)
	require.InDelta(t, 50, found.AvailabilityPct, 0.001, "availability must weight minute buckets, not request samples")
}
