package service

import (
	"context"
	"time"
)

type OAuthGroupAvailability struct {
	GroupID           int64  `json:"group_id"`
	GroupName         string `json:"group_name"`
	AvailableAccounts int64  `json:"available_accounts"`
}

type OAuthGroupAvailabilityReport struct {
	QueriedAt time.Time                `json:"queried_at"`
	Groups    []OAuthGroupAvailability `json:"groups"`
}

// GroupAvailability reads scheduling state only. It neither probes upstreams nor
// saves health reports, and remains available when Ops monitoring is disabled.
func (s *OAuthHealthService) GroupAvailability(ctx context.Context) (*OAuthGroupAvailabilityReport, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	now := s.now().UTC()
	groups, err := s.repo.GetOAuthGroupAvailability(ctx, now)
	if err != nil {
		return nil, err
	}
	if groups == nil {
		groups = []OAuthGroupAvailability{}
	}
	return &OAuthGroupAvailabilityReport{QueriedAt: now, Groups: groups}, nil
}
