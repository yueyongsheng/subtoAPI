//go:build unit

package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestOAuthQualityCustomPromptReachesNativeProtocols(t *testing.T) {
	for _, test := range []struct {
		model, path, promptPath string
		response                func() *http.Response
	}{
		{"deepseek-v4-flash", "/chat/completions", "messages.0.content", adaptiveCNChatTestResponse},
		{"grok-4.6", "/responses", "input.0.content.0.text", adaptiveCNResponsesTestResponse},
		{"minimax-m3", "/messages", "messages.0.content.0.text", adaptiveCNAnthropicTestResponse},
	} {
		t.Run(test.model, func(t *testing.T) {
			account := openCodeGoTestAccount(401)
			svc, upstream := adaptiveCNAccountTestService(account, test.response())
			result, err := svc.RunTestBackgroundWithPrompt(context.Background(), account.ID, test.model, "CUSTOM QUESTION 123")
			require.NoError(t, err)
			require.Equal(t, "success", result.Status)
			require.Len(t, upstream.requests, 1)
			require.Contains(t, upstream.requests[0].URL.Path, test.path)
			require.Equal(t, "CUSTOM QUESTION 123", gjson.GetBytes(upstream.lastBody, test.promptPath).String())
		})
	}
}

func TestOAuthQualityAdaptiveUsesOneModelRequestPerQuestion(t *testing.T) {
	account := adaptiveCNAccountTestAccount(402, PlatformDeepseek)
	svc, upstream := adaptiveCNAccountTestService(account, adaptiveCNChatTestResponse())
	result, err := svc.RunTestBackgroundWithPrompt(context.Background(), account.ID, "deepseek-v4-flash", "CUSTOM QUESTION")
	require.NoError(t, err)
	require.Equal(t, "success", result.Status)
	require.Len(t, upstream.requests, 1)
	require.Equal(t, "CUSTOM QUESTION", gjson.GetBytes(upstream.lastBody, "messages.0.content").String())
}

func TestOAuthQualityAntigravityPromptSurvivesTransform(t *testing.T) {
	svc := &AntigravityGatewayService{}
	for _, model := range []string{"gemini-2.5-flash", "claude-sonnet-4-5"} {
		t.Run(model, func(t *testing.T) {
			build := svc.buildGeminiTestRequest
			if model == "claude-sonnet-4-5" {
				build = svc.buildClaudeTestRequest
			}
			body, err := build("fixture-project", model, "CUSTOM question with \"quotes\"\nand newline")
			require.NoError(t, err)
			require.Contains(t, gjson.GetBytes(body, "request.contents").String(), "CUSTOM question")
			require.GreaterOrEqual(t, gjson.GetBytes(body, "request.generationConfig.maxOutputTokens").Int(), int64(4096))
			body, err = build("fixture-project", model)
			require.NoError(t, err)
			require.NotContains(t, string(body), "CUSTOM question")
		})
	}
}
