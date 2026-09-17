package admin

import (
	"context"
	"net/http"
	"sort"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *UserHandler) GetSpendingRanking(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	ranking, err := h.userService.GetAdminSpendingRanking(ctx, time.Now())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, ranking)
}

func (h *UserHandler) GetConcurrencyRanking(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	ranking := service.AdminConcurrencyRanking{QueriedAt: time.Now().UTC(), Users: []service.AdminUserConcurrency{}}
	ids, err := h.userService.GetAdminRankingUserIDs(ctx)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if h.concurrencyService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Concurrency data is temporarily unavailable")
		return
	}
	for start := 0; start < len(ids); start += 500 {
		users := make([]service.UserWithConcurrency, 0, min(500, len(ids)-start))
		for _, id := range ids[start:min(start+500, len(ids))] {
			users = append(users, service.UserWithConcurrency{ID: id})
		}
		loads, err := h.concurrencyService.GetUsersLoadBatch(ctx, users)
		if err != nil {
			response.Error(c, http.StatusServiceUnavailable, "Concurrency data is temporarily unavailable")
			return
		}
		for _, user := range users {
			load := loads[user.ID]
			if load == nil {
				response.Error(c, http.StatusServiceUnavailable, "Concurrency data is temporarily unavailable")
				return
			}
			if load.CurrentConcurrency > 0 {
				ranking.Users = append(ranking.Users, service.AdminUserConcurrency{UserID: user.ID, CurrentConcurrency: int64(load.CurrentConcurrency)})
			}
		}
		// Bound retained results even on installations with many active users.
		sort.Slice(ranking.Users, func(i, j int) bool {
			a, b := ranking.Users[i], ranking.Users[j]
			if a.CurrentConcurrency == b.CurrentConcurrency {
				return a.UserID < b.UserID
			}
			return a.CurrentConcurrency > b.CurrentConcurrency
		})
		if len(ranking.Users) > service.AdminUserRankingLimit {
			ranking.Users = ranking.Users[:service.AdminUserRankingLimit]
		}
	}
	if len(ranking.Users) > 0 {
		topIDs := make([]int64, len(ranking.Users))
		for i, user := range ranking.Users {
			topIDs[i] = user.UserID
		}
		// Preserve counts and IDs if the optional identity lookup fails.
		if identities, err := h.userService.GetAdminRankingIdentities(ctx, topIDs); err == nil {
			for i := range ranking.Users {
				ranking.Users[i].User = identities[ranking.Users[i].UserID]
			}
		}
	}
	response.Success(c, ranking)
}
