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
	start, end time.Time
}

func (r *overviewUserRepo) GetAdminUserOverview(_ context.Context, start, end, _ time.Time) (*service.AdminUserOverview, []int64, error) {
	r.start, r.end = start, end
	ids := make([]int64, 1001)
	for i := range ids {
		ids[i] = int64(i + 1)
	}
	return &service.AdminUserOverview{TotalUsers: 1001, PositiveBalanceUsers: 900, TotalBalance: 240073.05, ActiveUsers10m: 36, TodayUserCost: 250.5}, ids, nil
}

type overviewConcurrencyCache struct {
	service.ConcurrencyCache
	seen int
	fail bool
}

func (c *overviewConcurrencyCache) GetUsersLoadBatch(_ context.Context, users []service.UserWithConcurrency) (map[int64]*service.UserLoadInfo, error) {
	if c.fail {
		return nil, errors.New("redis unavailable")
	}
	c.seen += len(users)
	loads := make(map[int64]*service.UserLoadInfo, len(users))
	for _, user := range users {
		loads[user.ID] = &service.UserLoadInfo{CurrentConcurrency: 2, WaitingCount: 100}
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
			} else {
				require.Equal(t, 1001, cache.seen)
				require.Equal(t, int64(2002), *payload.Data.CurrentConcurrency)
				require.Equal(t, int64(2), *payload.Data.MaxUserConcurrency)
			}
		})
	}
}
