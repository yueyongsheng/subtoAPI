package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type qualityHandlerRepo struct {
	group *int64
	types []string
}

func (r *qualityHandlerRepo) ListOAuthQualityAccounts(_ context.Context, group *int64, types []string) ([]service.OAuthQualityAccount, error) {
	r.group, r.types = group, types
	return []service.OAuthQualityAccount{{ID: 2, Name: "fixture", Type: "apikey", GroupIDs: []int64{7}}}, nil
}

type qualityHandlerRunner struct {
	calls  int
	prompt string
}

func (r *qualityHandlerRunner) RunTestBackgroundWithPrompt(_ context.Context, _ int64, _, prompt string) (*service.ScheduledTestResult, error) {
	r.calls++
	r.prompt = prompt
	return &service.ScheduledTestResult{Status: "success", ResponseText: "7"}, nil
}
func TestOAuthQualityHandlersForwardSelectionAndRejectInvalidCustom(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo, runner := &qualityHandlerRepo{}, &qualityHandlerRunner{}
	h := &AccountHandler{}
	h.SetOAuthQualityService(service.NewOAuthQualityService(repo, runner, nil))
	router := gin.New()
	router.POST("/accounts", h.ListOAuthQualityAccounts)
	router.POST("/run", h.RunOAuthQuality)
	send := func(path, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		return w
	}
	w := send("/accounts", `{"group_id":7,"account_types":["oauth","apikey"]}`)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, int64(7), *repo.group)
	require.Equal(t, []string{"oauth", "apikey"}, repo.types)
	w = send("/run", `{"group_id":7,"account_ids":[2],"account_types":["apikey"],"model_id":"gpt-6-astra","probe_keys":["custom"],"custom_probe":{"prompt":"days in a week?","expected":"7"}}`)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "days in a week?", runner.prompt)
	require.Equal(t, 1, runner.calls)
	for _, body := range []string{
		`{"account_ids":[2],"account_types":[],"model_id":"gpt-6-astra","probe_keys":["english_knowledge"]}`,
		`{"account_ids":[2],"account_types":["apikey"],"model_id":"gpt-6-astra","probe_keys":["custom"]}`,
		`{"account_ids":[2],"account_types":["apikey"],"model_id":"gpt-6-astra","probe_keys":[]}`,
	} {
		require.Equal(t, http.StatusBadRequest, send("/run", body).Code)
	}
	require.Equal(t, 1, runner.calls)
}
