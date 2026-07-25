package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ChannelHealthObservation is the metadata-only outcome emitted by the gateway.
// It intentionally carries no request or response content.
type ChannelHealthObservation struct {
	GroupID           int64
	APIKeyID          int64
	Model             string
	Success           bool
	RecoveredError    bool
	QualifyingFailure bool
	Duration          time.Duration
	ObservedAt        time.Time
}

type channelObserverRouteKey struct {
	groupID int64
	model   string
}

type channelObserverTarget struct {
	monitorID int64
	model     string
}

type channelObserverEvent struct {
	target         channelObserverTarget
	success        bool
	recoveredError bool
	failure        bool
	slow           bool
	durationMs     int
	observedAt     time.Time
}

type channelObserverBucketKey struct {
	monitorID  int64
	model      string
	bucketUnix int64
}

type channelObserverBucket struct {
	samples         int
	successes       int
	failures        int
	recoveredErrors int
	slow            int
	durationTotalMs int64
	durationCount   int
}

type channelMonitorObserver struct {
	mu        sync.RWMutex
	routes    map[channelObserverRouteKey][]channelObserverTarget
	probeKeys map[int64]struct{}
	queue     chan channelObserverEvent
	stopCh    chan struct{}
	started   bool
	stopOnce  sync.Once
	wg        sync.WaitGroup
	dropped   atomic.Uint64
}

func (o *channelMonitorObserver) init() {
	o.routes = make(map[channelObserverRouteKey][]channelObserverTarget)
	o.probeKeys = make(map[int64]struct{})
	o.queue = make(chan channelObserverEvent, monitorTrafficQueueSize)
	o.stopCh = make(chan struct{})
}

func (s *ChannelMonitorService) StartTrafficObserver(ctx context.Context) {
	if s == nil {
		return
	}
	s.refreshTrafficObserverIndex(ctx)
	o := &s.observer
	o.mu.Lock()
	if o.started {
		o.mu.Unlock()
		return
	}
	o.started = true
	o.wg.Add(1)
	o.mu.Unlock()
	go s.runTrafficObserver()
}

func (s *ChannelMonitorService) StopTrafficObserver() {
	if s == nil {
		return
	}
	o := &s.observer
	o.stopOnce.Do(func() { close(o.stopCh) })
	o.wg.Wait()
}

func (s *ChannelMonitorService) refreshTrafficObserverIndex(ctx context.Context) {
	if s == nil || s.repo == nil {
		return
	}
	monitors, err := s.repo.ListEnabled(ctx)
	if err != nil {
		slog.Warn("channel_monitor: refresh traffic observer index failed", "error", err)
		return
	}
	routes := make(map[channelObserverRouteKey][]channelObserverTarget)
	probeKeys := make(map[int64]struct{})
	for _, m := range monitors {
		if m.Mode != MonitorModeHybrid || m.GroupID == nil || *m.GroupID <= 0 {
			continue
		}
		if m.ProbeAPIKeyID != nil && *m.ProbeAPIKeyID > 0 {
			probeKeys[*m.ProbeAPIKeyID] = struct{}{}
		}
		models := append([]string{m.PrimaryModel}, m.ExtraModels...)
		for _, model := range models {
			model = strings.TrimSpace(model)
			if model == "" {
				continue
			}
			key := channelObserverRouteKey{groupID: *m.GroupID, model: model}
			routes[key] = append(routes[key], channelObserverTarget{monitorID: m.ID, model: model})
		}
	}
	s.observer.mu.Lock()
	s.observer.routes = routes
	s.observer.probeKeys = probeKeys
	s.observer.mu.Unlock()
}

// ObserveChannelHealth is deliberately non-blocking. Monitoring loss is preferable
// to adding latency or failure to a customer request.
func (s *ChannelMonitorService) ObserveChannelHealth(observation ChannelHealthObservation) {
	if s == nil || observation.GroupID <= 0 || observation.APIKeyID <= 0 {
		return
	}
	model := strings.TrimSpace(observation.Model)
	if model == "" || (!observation.Success && !observation.QualifyingFailure) {
		return
	}
	o := &s.observer
	o.mu.RLock()
	if _, excluded := o.probeKeys[observation.APIKeyID]; excluded {
		o.mu.RUnlock()
		return
	}
	targets := append([]channelObserverTarget(nil), o.routes[channelObserverRouteKey{groupID: observation.GroupID, model: model}]...)
	o.mu.RUnlock()
	if len(targets) == 0 {
		return
	}
	observedAt := observation.ObservedAt
	if observedAt.IsZero() {
		observedAt = time.Now()
	}
	for _, target := range targets {
		event := channelObserverEvent{
			target:         target,
			success:        observation.Success,
			recoveredError: observation.RecoveredError,
			failure:        observation.QualifyingFailure,
			slow:           observation.Success && observation.Duration >= monitorDegradedThreshold,
			durationMs:     int(observation.Duration / time.Millisecond),
			observedAt:     observedAt,
		}
		select {
		case o.queue <- event:
		default:
			o.dropped.Add(1)
		}
	}
}

func (s *ChannelMonitorService) runTrafficObserver() {
	defer s.observer.wg.Done()
	ticker := time.NewTicker(monitorTrafficBucketInterval)
	defer ticker.Stop()
	buckets := make(map[channelObserverBucketKey]*channelObserverBucket)
	for {
		select {
		case event := <-s.observer.queue:
			addChannelObserverEvent(buckets, event)
		case now := <-ticker.C:
			s.flushTrafficBuckets(buckets, now.UTC().Truncate(monitorTrafficBucketInterval), false)
		case <-s.observer.stopCh:
			for {
				select {
				case event := <-s.observer.queue:
					addChannelObserverEvent(buckets, event)
				default:
					s.flushTrafficBuckets(buckets, time.Now().UTC().Add(monitorTrafficBucketInterval), true)
					return
				}
			}
		}
	}
}

func addChannelObserverEvent(buckets map[channelObserverBucketKey]*channelObserverBucket, event channelObserverEvent) {
	bucketStart := event.observedAt.UTC().Truncate(monitorTrafficBucketInterval)
	key := channelObserverBucketKey{monitorID: event.target.monitorID, model: event.target.model, bucketUnix: bucketStart.Unix()}
	bucket := buckets[key]
	if bucket == nil {
		bucket = &channelObserverBucket{}
		buckets[key] = bucket
	}
	bucket.samples++
	if event.success {
		bucket.successes++
		if event.durationMs >= 0 {
			bucket.durationTotalMs += int64(event.durationMs)
			bucket.durationCount++
		}
	}
	if event.failure {
		bucket.failures++
	}
	if event.recoveredError {
		bucket.recoveredErrors++
	}
	if event.slow {
		bucket.slow++
	}
}

func (s *ChannelMonitorService) flushTrafficBuckets(buckets map[channelObserverBucketKey]*channelObserverBucket, before time.Time, force bool) {
	for key, bucket := range buckets {
		bucketStart := time.Unix(key.bucketUnix, 0).UTC()
		if !force && !bucketStart.Before(before) {
			continue
		}
		if err := s.persistTrafficBucket(key, bucketStart, bucket); err != nil {
			slog.Warn("channel_monitor: persist traffic bucket failed", "monitor_id", key.monitorID, "model", key.model, "error", err)
			continue
		}
		delete(buckets, key)
	}
}

func (s *ChannelMonitorService) persistTrafficBucket(key channelObserverBucketKey, bucketStart time.Time, bucket *channelObserverBucket) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var previous *ChannelMonitorHistoryEntry
	if bucket.successes == 0 && bucket.failures > 0 {
		history, err := s.repo.ListHistory(ctx, key.monitorID, key.model, 1)
		if err != nil {
			return err
		}
		if len(history) > 0 {
			previous = history[0]
		}
	}
	status := trafficBucketStatus(bucket, previous)
	var latency *int
	if bucket.durationCount > 0 {
		value := int(bucket.durationTotalMs / int64(bucket.durationCount))
		latency = &value
	}
	checkedAt := bucketStart.Add(monitorTrafficBucketInterval)
	message := fmt.Sprintf("real traffic: samples=%d success=%d failure=%d recovered=%d slow=%d", bucket.samples, bucket.successes, bucket.failures, bucket.recoveredErrors, bucket.slow)
	row := &ChannelMonitorHistoryRow{
		MonitorID: key.monitorID, Model: key.model, Status: status,
		Source: MonitorSourceRealTraffic, BucketStart: &bucketStart,
		SampleCount: bucket.samples, SuccessCount: bucket.successes, FailureCount: bucket.failures,
		RecoveredErrorCount: bucket.recoveredErrors, SlowCount: bucket.slow,
		LatencyMs: latency, Message: truncateMessage(message), CheckedAt: checkedAt,
	}
	if err := s.repo.UpsertTrafficBucket(ctx, row); err != nil {
		return err
	}
	return s.repo.MarkChecked(ctx, key.monitorID, checkedAt)
}

func trafficBucketStatus(bucket *channelObserverBucket, previous *ChannelMonitorHistoryEntry) string {
	if bucket == nil {
		return MonitorStatusOperational
	}
	if bucket.successes > 0 {
		if bucket.failures > 0 || bucket.recoveredErrors > 0 || bucket.slow > 0 {
			return MonitorStatusDegraded
		}
		return MonitorStatusOperational
	}
	if bucket.failures > 0 {
		if previous != nil && previous.Source == MonitorSourceRealTraffic && previous.SuccessCount == 0 && previous.FailureCount > 0 {
			return MonitorStatusError
		}
		return MonitorStatusDegraded
	}
	return MonitorStatusOperational
}
