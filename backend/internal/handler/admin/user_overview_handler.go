package admin

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// GetOverview serves a fresh snapshot only when the admin page requests it.
func (h *UserHandler) GetOverview(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	stats, ids, err := h.userService.GetAdminUserOverview(ctx, time.Now())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	// A missing or failed Redis read stays null instead of displaying a false zero.
	if h.concurrencyService != nil {
		var total int64
		var maximum int64
		var maximumUserID, maximumUserCount int64
		available := true
		for start := 0; start < len(ids); start += 500 {
			end := min(start+500, len(ids))
			users := make([]service.UserWithConcurrency, 0, end-start)
			for _, id := range ids[start:end] {
				users = append(users, service.UserWithConcurrency{ID: id})
			}
			loads, loadErr := h.concurrencyService.GetUsersLoadBatch(ctx, users)
			if loadErr != nil {
				available = false
				break
			}
			for _, user := range users {
				load := loads[user.ID]
				if load == nil {
					available = false
					break
				}
				current := int64(max(0, load.CurrentConcurrency))
				total += current
				if current > maximum {
					maximum, maximumUserID, maximumUserCount = current, user.ID, 1
				} else if current > 0 && current == maximum {
					maximumUserCount++
					maximumUserID = min(maximumUserID, user.ID)
				}
			}
			if !available {
				break
			}
		}
		if available {
			stats.CurrentConcurrency = &total
			stats.MaxUserConcurrency = &maximum
			stats.MaxConcurrencyUserCount = maximumUserCount
			if maximum > 0 {
				// Identity failure must not discard the successfully read concurrency.
				if user, err := h.userService.GetAdminOverviewUser(ctx, maximumUserID); err == nil {
					stats.MaxConcurrencyUser = user
				}
			}
		}
	}
	response.Success(c, stats)
}
