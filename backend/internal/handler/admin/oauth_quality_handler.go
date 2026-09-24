package admin

import (
	"errors"
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// SetOAuthQualityService attaches the explicit, administrator-triggered model
// quality probe service without changing the account handler constructor.
func (h *AccountHandler) SetOAuthQualityService(s *service.OAuthQualityService) {
	h.oauthQuality = s
}

type oauthQualityScopeRequest struct {
	GroupID      *int64   `json:"group_id"`
	AccountTypes []string `json:"account_types"`
}

type oauthQualityRunRequest struct {
	GroupID      *int64                           `json:"group_id"`
	AccountIDs   []int64                          `json:"account_ids"`
	ModelID      string                           `json:"model_id"`
	AccountTypes []string                         `json:"account_types"`
	ProbeKeys    []string                         `json:"probe_keys"`
	CustomProbe  *service.OAuthQualityCustomProbe `json:"custom_probe"`
}

func validOAuthQualityGroup(groupID *int64) bool {
	return groupID == nil || *groupID == -1 || *groupID > 0
}

func (h *AccountHandler) ListOAuthQualityAccounts(c *gin.Context) {
	if h.oauthQuality == nil {
		response.Error(c, http.StatusServiceUnavailable, "OAuth quality service unavailable")
		return
	}
	var req oauthQualityScopeRequest
	if err := c.ShouldBindJSON(&req); err != nil || !validOAuthQualityGroup(req.GroupID) || !service.ValidQualityTypes(req.AccountTypes) {
		response.BadRequest(c, "Invalid OAuth quality scope")
		return
	}
	accounts, err := h.oauthQuality.List(c.Request.Context(), req.GroupID, req.AccountTypes)
	if err != nil {
		response.Error(c, http.StatusServiceUnavailable, "OAuth quality account query failed")
		return
	}
	response.Success(c, gin.H{"accounts": accounts})
}

func (h *AccountHandler) RunOAuthQuality(c *gin.Context) {
	if h.oauthQuality == nil {
		response.Error(c, http.StatusServiceUnavailable, "OAuth quality service unavailable")
		return
	}
	var req oauthQualityRunRequest
	if err := c.ShouldBindJSON(&req); err != nil || !validOAuthQualityGroup(req.GroupID) || !service.ValidQualityTypes(req.AccountTypes) {
		response.BadRequest(c, "Invalid OAuth quality request")
		return
	}
	if len(req.AccountIDs) == 0 || len(req.AccountIDs) > 50 {
		response.BadRequest(c, "Select between 1 and 50 accounts")
		return
	}
	report, err := h.oauthQuality.Run(c.Request.Context(), req.GroupID, req.AccountIDs, req.ModelID, req.AccountTypes, req.ProbeKeys, req.CustomProbe)
	if err != nil {
		if errors.Is(err, service.ErrInvalidQualitySelection) {
			response.BadRequest(c, "Invalid selection; reload accounts and select at least one valid check")
			return
		}
		response.Error(c, http.StatusServiceUnavailable, "OAuth quality check failed; refresh and retry")
		return
	}
	response.Success(c, report)
}
