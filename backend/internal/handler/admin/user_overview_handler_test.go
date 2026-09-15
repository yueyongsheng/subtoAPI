package admin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type overviewUserRepo struct {
	service.UserRepository
	start, end, todayStart time.Time
	identityIDs            []int64
	identityFail           bool
}

func (r *overviewUserRepo) GetAdminOverviewUser(_ context.Context, id int64) (*service.AdminOverviewUser, error) {
	r.identityIDs = append(r.identityIDs, id)
	if r.identityFail {
		return nil, errors.New("identity unavailable")
	}
	return &service.AdminOverviewUser{ID: id, Username: "example-user", Email: "example@example.test"}, nil
}

func (r *overviewUserRepo) GetAdminUserOverview(_ context.Context, start, end, todayStart time.Time) (*service.AdminUserOverview, []int64, error) {
	r.start, r.end, r.todayStart = start, end, todayStart
	ids := make([]int64, 1001)
	for i := range ids {
		ids[i] = int64(i + 1)
	}
	return &service.AdminUserOverview{TotalUsers: 1001, PositiveBalanceUsers: 900, TotalBalance: 240073.05, ActiveUsers10m: 36, TodayUserCost: 250.5}, ids, nil
}

type overviewConcurrencyCache struct {
	service.ConcurrencyCache
	seen      int
	fail      bool
	occupancy map[int64]int
}

func (c *overviewConcurrencyCache) GetUsersLoadBatch(_ context.Context, users []service.UserWithConcurrency) (map[int64]*service.UserLoadInfo, error) {
	if c.fail {
		return nil, errors.New("redis unavailable")
	}
	c.seen += len(users)
	loads := make(map[int64]*service.UserLoadInfo, len(users))
	for _, user := range users {
		loads[user.ID] = &service.UserLoadInfo{CurrentConcurrency: 2, WaitingCount: 100}
		if c.occupancy != nil {
			loads[user.ID].CurrentConcurrency = c.occupancy[user.ID]
			continue
		}
		switch user.ID {
		case 750:
			loads[user.ID].CurrentConcurrency = 14
		case 1001:
			loads[user.ID].CurrentConcurrency = 5
		}
	}
	return loads, nil
}

func TestUserOverview_AllUsersAndUnavailableConcurrency(t *testing.T) {
	for _, failed := range []bool{false, true} {
		t.Run(map[bool]string{false: "current occupancy across all users", true: "redis failure stays unknown"}[failed], func(t *testing.T) {
			repo := &overviewUserRepo{}
			cache := &overviewConcurrencyCache{fail: failed}
			h := &UserHandler{userService: service.NewUserService(repo, nil, nil, nil), concurrencyService: service.NewConcurrencyService(cache)}
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/api/v1/admin/users/overview?search=one-user&page=20", nil)
			h.GetOverview(c)
			require.Equal(t, 200, w.Code)
			require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
			var payload struct {
				Data service.AdminUserOverview `json:"data"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
			require.Equal(t, int64(1001), payload.Data.TotalUsers)
			require.InDelta(t, 9602.92, payload.Data.BalanceCNY, 0.000001)
			require.InDelta(t, 10.02, payload.Data.TodayUserCostCNY, 0.000001)
			require.Equal(t, 10*time.Minute, repo.end.Sub(repo.start))
			require.True(t, payload.Data.QueriedAt.Equal(repo.end))
			if failed {
				require.Nil(t, payload.Data.CurrentConcurrency)
				require.Nil(t, payload.Data.MaxUserConcurrency)
				require.Nil(t, payload.Data.MaxConcurrencyUser)
				require.Empty(t, repo.identityIDs)
			} else {
				require.Equal(t, 1001, cache.seen)
				require.Equal(t, int64(2017), *payload.Data.CurrentConcurrency)
				require.Equal(t, int64(14), *payload.Data.MaxUserConcurrency)
				require.Equal(t, int64(750), payload.Data.MaxConcurrencyUser.ID)
				require.Equal(t, "example-user", payload.Data.MaxConcurrencyUser.Username)
				require.Equal(t, int64(1), payload.Data.MaxConcurrencyUserCount)
				require.Equal(t, []int64{750}, repo.identityIDs)
			}
		})
	}
}

func TestUserOverview_MaximumIdentityBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		occupancy             map[int64]int
		identityFail          bool
		maximum, userID, tied int64
	}{
		{"tie across batches chooses lowest ID", map[int64]int{750: 14, 1001: 14}, false, 14, 750, 2},
		{"later higher value resets tie", map[int64]int{1: 2, 2: 2, 1001: 14}, false, 14, 1001, 1},
		{"zero and negative occupancy has no winner", map[int64]int{1: -1}, false, 0, 0, 0},
		{"identity failure retains numbers", map[int64]int{750: 14}, true, 14, 750, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &overviewUserRepo{identityFail: tc.identityFail}
			cache := &overviewConcurrencyCache{occupancy: tc.occupancy}
			h := &UserHandler{userService: service.NewUserService(repo, nil, nil, nil), concurrencyService: service.NewConcurrencyService(cache)}
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/api/v1/admin/users/overview", nil)
			h.GetOverview(c)
			require.Equal(t, 200, w.Code)
			var payload struct {
				Data service.AdminUserOverview `json:"data"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
			require.Equal(t, tc.maximum, *payload.Data.MaxUserConcurrency)
			require.Equal(t, tc.tied, payload.Data.MaxConcurrencyUserCount)
			if tc.maximum == 0 {
				require.Empty(t, repo.identityIDs)
			} else {
				require.Equal(t, []int64{tc.userID}, repo.identityIDs)
			}
			if tc.identityFail || tc.maximum == 0 {
				require.Nil(t, payload.Data.MaxConcurrencyUser)
			} else {
				require.Equal(t, tc.userID, payload.Data.MaxConcurrencyUser.ID)
			}
		})
	}
}

func TestUserOverview_TodayStartsAtUTC8Midnight(t *testing.T) {
	for _, tc := range []struct{ queriedAt, expectedStart string }{
		{"2026-09-15T15:59:59Z", "2026-09-14T16:00:00Z"},
		{"2026-09-15T16:00:00Z", "2026-09-15T16:00:00Z"},
		{"2026-09-16T00:05:00+08:00", "2026-09-15T16:00:00Z"},
	} {
		t.Run(tc.queriedAt, func(t *testing.T) {
			repo := &overviewUserRepo{}
			queriedAt, err := time.Parse(time.RFC3339, tc.queriedAt)
			require.NoError(t, err)
			stats, _, err := service.NewUserService(repo, nil, nil, nil).GetAdminUserOverview(context.Background(), queriedAt)
			require.NoError(t, err)
			require.Equal(t, tc.expectedStart, repo.todayStart.Format(time.RFC3339))
			require.True(t, queriedAt.Equal(repo.end))
			require.Equal(t, 10*time.Minute, repo.end.Sub(repo.start))
			require.InDelta(t, 10.02, stats.TodayUserCostCNY, 0.000001)
		})
	}
}
