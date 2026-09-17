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

type rankingUserRepo struct {
	service.UserRepository
	ids          []int64
	identityIDs  []int64
	identityFail bool
	start, end   time.Time
}

func (r *rankingUserRepo) GetAdminRankingUserIDs(context.Context) ([]int64, error) { return r.ids, nil }
func (r *rankingUserRepo) GetAdminRankingIdentities(_ context.Context, ids []int64) (map[int64]*service.AdminOverviewUser, error) {
	r.identityIDs = append(r.identityIDs, ids...)
	if r.identityFail {
		return nil, errors.New("identity unavailable")
	}
	result := map[int64]*service.AdminOverviewUser{}
	for _, id := range ids {
		result[id] = &service.AdminOverviewUser{ID: id, Username: "Example"}
	}
	return result, nil
}
func (r *rankingUserRepo) GetAdminSpendingRanking(_ context.Context, start, end time.Time) ([]service.AdminUserSpending, error) {
	r.start, r.end = start, end
	return []service.AdminUserSpending{{UserID: 12, Cost: 1234.56}}, nil
}

func TestUserRankings_ConcurrencyTopTenAcrossBatches(t *testing.T) {
	repo := &rankingUserRepo{}
	for id := int64(1); id <= 1001; id++ {
		repo.ids = append(repo.ids, id)
	}
	cache := &overviewConcurrencyCache{occupancy: map[int64]int{1001: 30, 501: 20, 1: -1}}
	for id := int64(10); id <= 22; id++ {
		cache.occupancy[id] = 10
	}
	h := &UserHandler{userService: service.NewUserService(repo, nil, nil, nil), concurrencyService: service.NewConcurrencyService(cache)}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/admin/users/overview/concurrency-ranking?search=x&limit=999", nil)
	h.GetConcurrencyRanking(c)
	require.Equal(t, 200, w.Code)
	require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	var payload struct {
		Data service.AdminConcurrencyRanking `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
	require.Len(t, payload.Data.Users, 10)
	want := []int64{1001, 501, 10, 11, 12, 13, 14, 15, 16, 17}
	require.Equal(t, want, repo.identityIDs)
	for i, user := range payload.Data.Users {
		require.Equal(t, want[i], user.UserID)
		require.Equal(t, user.UserID, user.User.ID)
		require.Positive(t, user.CurrentConcurrency)
	}
	require.Equal(t, 1001, cache.seen)
}

type rankingMissingLoadCache struct{ service.ConcurrencyCache }

func (*rankingMissingLoadCache) GetUsersLoadBatch(context.Context, []service.UserWithConcurrency) (map[int64]*service.UserLoadInfo, error) {
	return map[int64]*service.UserLoadInfo{}, nil
}

func TestUserRankings_ConcurrencyEmptyPartialAndFailures(t *testing.T) {
	for _, tc := range []struct {
		name                        string
		occupancy                   map[int64]int
		fail, missing, identityFail bool
		status, count               int
	}{
		{"no active users", map[int64]int{}, false, false, false, 200, 0},
		{"two active users", map[int64]int{1: 7, 2: 2}, false, false, false, 200, 2},
		{"redis failure", nil, true, false, false, 503, 0},
		{"missing load is not zero", nil, false, true, false, 503, 0},
		{"identity failure keeps occupancy", map[int64]int{2: 3}, false, false, true, 200, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &rankingUserRepo{ids: []int64{1, 2, 3}, identityFail: tc.identityFail}
			var cache service.ConcurrencyCache = &overviewConcurrencyCache{fail: tc.fail, occupancy: tc.occupancy}
			if tc.missing {
				cache = &rankingMissingLoadCache{}
			}
			h := &UserHandler{userService: service.NewUserService(repo, nil, nil, nil), concurrencyService: service.NewConcurrencyService(cache)}
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/api/v1/admin/users/overview/concurrency-ranking", nil)
			h.GetConcurrencyRanking(c)
			require.Equal(t, tc.status, w.Code)
			if tc.status != 200 {
				require.Empty(t, repo.identityIDs)
				return
			}
			var payload struct {
				Data service.AdminConcurrencyRanking `json:"data"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
			require.Len(t, payload.Data.Users, tc.count)
			require.NotNil(t, payload.Data.Users)
			if tc.identityFail {
				require.Nil(t, payload.Data.Users[0].User)
				require.Equal(t, int64(3), payload.Data.Users[0].CurrentConcurrency)
			}
		})
	}
}

func TestUserRankings_SpendingUsesUTC8DayAndCNYConversion(t *testing.T) {
	for _, tc := range []struct{ at, start string }{
		{"2026-09-17T15:59:59Z", "2026-09-16T16:00:00Z"},
		{"2026-09-17T16:00:00Z", "2026-09-17T16:00:00Z"},
	} {
		repo := &rankingUserRepo{}
		at, err := time.Parse(time.RFC3339, tc.at)
		require.NoError(t, err)
		ranking, err := service.NewUserService(repo, nil, nil, nil).GetAdminSpendingRanking(context.Background(), at)
		require.NoError(t, err)
		require.Equal(t, tc.start, repo.start.Format(time.RFC3339))
		require.True(t, repo.end.Equal(at))
		require.InDelta(t, 49.38, ranking.Users[0].CostCNY, 0.000001)
	}
}
