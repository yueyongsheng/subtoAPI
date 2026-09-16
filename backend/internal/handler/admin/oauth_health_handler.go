package admin

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *AccountHandler) SetOAuthHealthService(s *service.OAuthHealthService) { h.oauthHealth = s }

type oauthHealthRequest struct {
	GroupID  *int64                         `json:"group_id"`
	Accounts []service.OAuthHealthSelection `json:"accounts"`
	Restore  bool                           `json:"restore"`
}

func (h *AccountHandler) CheckOAuthHealth(c *gin.Context)  { h.runOAuthHealth(c, false) }
func (h *AccountHandler) AdjustOAuthHealth(c *gin.Context) { h.runOAuthHealth(c, true) }

func (h *AccountHandler) runOAuthHealth(c *gin.Context, change bool) {
	if h.oauthHealth == nil {
		response.Error(c, http.StatusServiceUnavailable, "OAuth health service unavailable")
		return
	}
	var req oauthHealthRequest
	if c.ShouldBindJSON(&req) != nil || (req.GroupID != nil && *req.GroupID != -1 && *req.GroupID <= 0) {
		response.BadRequest(c, "Invalid OAuth health scope")
		return
	}
	if change && (len(req.Accounts) == 0 || len(req.Accounts) > 2000) {
		response.BadRequest(c, "Invalid account selection")
		return
	}
	var result *service.OAuthHealthReport
	var err error
	if change {
		subject, ok := middleware.GetAuthSubjectFromContext(c)
		if !ok || subject.UserID <= 0 {
			response.Error(c, http.StatusUnauthorized, "Unauthorized")
			return
		}
		result, err = h.oauthHealth.Change(c.Request.Context(), req.GroupID, req.Accounts, req.Restore, subject.UserID)
	} else {
		result, err = h.oauthHealth.Check(c.Request.Context(), req.GroupID)
	}
	if err != nil {
		response.Error(c, http.StatusServiceUnavailable, "OAuth health check failed; refresh before retrying")
		return
	}
	response.Success(c, result)
}
