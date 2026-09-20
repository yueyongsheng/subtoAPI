package service

import (
	"testing"

	pluginv1 "github.com/Wei-Shaw/sub2api/pkg/pluginapi/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvaluatePluginCompatibility(t *testing.T) {
	manifest := testPluginManifest(nil)
	host := PluginHostInfo{Version: "0.1.179", BuildType: "release"}

	result := EvaluatePluginCompatibility(manifest, host)
	require.True(t, result.Compatible)
	assert.True(t, result.Tested)
	assert.Equal(t, "compatible", result.Status)

	manifest.Requires.TestedSub2APIVersions = []string{"0.1.178"}
	result = EvaluatePluginCompatibility(manifest, host)
	require.True(t, result.Compatible)
	assert.False(t, result.Tested)
	assert.Equal(t, "untested", result.Status)

	manifest.Requires.Sub2API = ">=0.2.0 <0.3.0"
	result = EvaluatePluginCompatibility(manifest, host)
	assert.False(t, result.Compatible)
	assert.Equal(t, "incompatible", result.Status)
}

func TestEvaluatePluginCompatibilityRejectsProtocolMismatch(t *testing.T) {
	manifest := testPluginManifest(nil)
	manifest.Requires.PluginProtocol = pluginv1.ProtocolVersion + 1

	result := EvaluatePluginCompatibility(manifest, PluginHostInfo{Version: "0.1.179"})

	assert.False(t, result.Compatible)
	assert.Equal(t, "incompatible", result.Status)
}

func TestMatchesSemverRange(t *testing.T) {
	assert.True(t, matchesSemverRange("0.1.179", ">=0.1.170, <0.2.0"))
	assert.True(t, matchesSemverRange("v1.2.3", "=1.2.3"))
	assert.False(t, matchesSemverRange("0.1.169", ">=0.1.170 <0.2.0"))
	assert.False(t, matchesSemverRange("dev", ">=0.1.0"))
	assert.False(t, matchesSemverRange("0.1.179", "^0.1.0"))
}

func TestPluginRevisionCompatibility(t *testing.T) {
	for _, tc := range []struct {
		version, bounds string
		want            bool
	}{
		{"0.2.7.5", ">=0.2.7 <0.3.0", true},
		{"v0.2.7.6", ">=0.2.7 <0.3.0", true},
		{"0.2.7.10", ">0.2.7.9 <=0.2.7.10", true},
		{"0.2.7.2", ">=0.2.7.10", false},
		{"0.2.7.999999999999999999999", ">0.2.7.99999999999999999999 <0.2.8", true},
		{"0.3.0.1", ">=0.2.7 <0.3.0", false},
		{"0.2.6.99", ">=0.2.7", false},
		{"0.2.7.05", ">=0.2.7", false},
		{"0.2.7.5.extra", ">=0.2.7", false},
		{"0.2.7.5-beta", ">=0.2.7", false},
		{"0.2.7-rc.1", ">=0.2.7", false},
		{"dev", ">=0.2.7", false},
		{"0.2.7.5", "^0.2.7", false},
	} {
		t.Run(tc.version+"/"+tc.bounds, func(t *testing.T) { require.Equal(t, tc.want, matchesSemverRange(tc.version, tc.bounds)) })
	}
	manifest := testPluginManifest(nil)
	manifest.Requires.Sub2API = ">=0.2.7 <0.3.0"
	manifest.Requires.TestedSub2APIVersions = []string{"0.2.7"}
	host := PluginHostInfo{Version: "0.2.7.6", BuildType: "release"}
	result := EvaluatePluginCompatibility(manifest, host)
	require.True(t, result.Compatible)
	require.False(t, result.Tested)
	require.Equal(t, "untested", result.Status)
	manifest.Requires.TestedSub2APIVersions = []string{"v0.2.7.6"}
	require.True(t, EvaluatePluginCompatibility(manifest, host).Tested)
	for _, mismatch := range []string{"protocol", "transport", "bridge"} {
		t.Run(mismatch, func(t *testing.T) {
			bad := manifest
			switch mismatch {
			case "protocol":
				bad.Requires.PluginProtocol++
			case "transport":
				bad.Requires.TransportAPI++
			case "bridge":
				bad.Requires.UIBridge++
			}
			require.False(t, EvaluatePluginCompatibility(bad, host).Compatible)
		})
	}
}
