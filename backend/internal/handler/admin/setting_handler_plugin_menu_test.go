package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestUpdateSettingsPluginMenuRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name   string
		stored bool
		body   map[string]any
		want   bool
	}{
		{"enable", false, map[string]any{"plugin_management_enabled": true}, true},
		{"disable", true, map[string]any{"plugin_management_enabled": false}, false},
		{"omitted_preserves_enabled", true, map[string]any{"site_name": "Gateway"}, true},
		{"omitted_preserves_disabled", false, map[string]any{"site_name": "Gateway"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, repo := newStepUpSwitchTestHandler(t, map[string]string{
				service.SettingKeyPluginManagementEnabled: strconv.FormatBool(tc.stored),
			})
			saved := doUpdateSettings(t, h, tc.body, nil)
			require.Equal(t, http.StatusOK, saved.Code)
			require.Equal(t, strconv.FormatBool(tc.want), repo.values[service.SettingKeyPluginManagementEnabled])

			assertFlag := func(rec *httptest.ResponseRecorder) {
				t.Helper()
				var envelope struct {
					Data map[string]any `json:"data"`
				}
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
				require.Equal(t, tc.want, envelope.Data["plugin_management_enabled"])
			}
			assertFlag(saved)
			loaded := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(loaded)
			c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings", nil)
			h.GetSettings(c)
			require.Equal(t, http.StatusOK, loaded.Code)
			assertFlag(loaded)
			public, err := h.settingService.GetPublicSettings(c.Request.Context())
			require.NoError(t, err)
			require.Equal(t, tc.want, public.PluginManagementEnabled)
			injected, err := h.settingService.GetPublicSettingsForInjection(c.Request.Context())
			require.NoError(t, err)
			payload, ok := injected.(*service.PublicSettingsInjectionPayload)
			require.True(t, ok)
			require.Equal(t, tc.want, payload.PluginManagementEnabled)
		})
	}
}

func TestDiffSettingsIncludesPluginMenuChanges(t *testing.T) {
	before := &service.SystemSettings{}
	after := &service.SystemSettings{PluginManagementEnabled: true}
	require.Contains(t, diffSettings(before, after, nil, nil, UpdateSettingsRequest{}), "plugin_management_enabled")
	require.Contains(t, diffSettings(after, before, nil, nil, UpdateSettingsRequest{}), "plugin_management_enabled")
	require.NotContains(t, diffSettings(after, after, nil, nil, UpdateSettingsRequest{}), "plugin_management_enabled")
}
