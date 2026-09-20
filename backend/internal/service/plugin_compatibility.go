package service

import (
	"fmt"
	"regexp"
	"strings"

	pluginv1 "github.com/Wei-Shaw/sub2api/pkg/pluginapi/v1"
	"golang.org/x/mod/semver"
)

type PluginHostInfo struct {
	Version   string
	BuildType string
}

func EvaluatePluginCompatibility(manifest PluginManifest, host PluginHostInfo) PluginCompatibility {
	result := PluginCompatibility{
		CurrentSub2API:     host.Version,
		RequiredSub2API:    manifest.Requires.Sub2API,
		RecommendedSub2API: manifest.Requires.RecommendedSub2APIVersion,
		PluginProtocol:     manifest.Requires.PluginProtocol,
		TransportAPI:       manifest.Requires.TransportAPI,
		UIBridge:           manifest.Requires.UIBridge,
	}
	if manifest.Requires.PluginProtocol != pluginv1.ProtocolVersion ||
		manifest.Requires.TransportAPI != pluginv1.TransportAPIVersion ||
		manifest.Requires.UIBridge != pluginv1.UIBridgeVersion {
		result.Status = "incompatible"
		result.Message = "插件协议版本与当前 Sub2API 不兼容"
		return result
	}
	if !matchesSemverRange(host.Version, manifest.Requires.Sub2API) {
		result.Status = "incompatible"
		result.Message = fmt.Sprintf("当前 Sub2API %s 不满足插件要求 %s", host.Version, manifest.Requires.Sub2API)
		return result
	}
	result.Compatible = true
	current, _ := parsePluginVersion(host.Version)
	for _, tested := range manifest.Requires.TestedSub2APIVersions {
		declared, valid := parsePluginVersion(tested)
		if valid && declared.identity == current.identity {
			result.Tested = true
			break
		}
	}
	if result.Tested {
		result.Status = "compatible"
		result.Message = "当前 Sub2API 版本已由插件声明测试"
	} else {
		result.Status = "untested"
		result.Message = "版本范围兼容，但插件未声明已测试当前 Sub2API 版本"
	}
	return result
}

// Local releases append a numeric revision to the upstream stable version.
// Keep that revision for ordering and exact tested-version matching.
var pluginRevisionVersion = regexp.MustCompile(`^v?((?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*))\.(0|[1-9][0-9]*)$`)

type pluginVersion struct {
	base, revision, identity string
}

func parsePluginVersion(raw string) (pluginVersion, bool) {
	if v := normalizeSemver(raw); v != "" {
		return pluginVersion{base: v, revision: "0", identity: v}, true
	}
	parts := pluginRevisionVersion.FindStringSubmatch(strings.TrimSpace(raw))
	if parts == nil {
		return pluginVersion{}, false
	}
	base := normalizeSemver(parts[1])
	if base == "" {
		return pluginVersion{}, false
	}
	return pluginVersion{base: base, revision: parts[2], identity: base + "." + parts[2]}, true
}

func comparePluginVersions(a, b pluginVersion) int {
	if compared := semver.Compare(a.base, b.base); compared != 0 {
		return compared
	}
	// Compare arbitrary-length numeric revisions without integer overflow.
	if len(a.revision) < len(b.revision) {
		return -1
	}
	if len(a.revision) > len(b.revision) {
		return 1
	}
	return strings.Compare(a.revision, b.revision)
}

func normalizeSemver(version string) string {
	v := strings.TrimSpace(version)
	if v == "" {
		return ""
	}
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	if !semver.IsValid(v) {
		return ""
	}
	return v
}

func matchesSemverRange(version, expression string) bool {
	v, valid := parsePluginVersion(version)
	if !valid {
		return false
	}
	tokens := strings.Fields(strings.ReplaceAll(expression, ",", " "))
	if len(tokens) == 0 {
		return false
	}
	for _, token := range tokens {
		op := "="
		raw := token
		for _, candidate := range []string{">=", "<=", ">", "<", "="} {
			if strings.HasPrefix(token, candidate) {
				op = candidate
				raw = strings.TrimSpace(strings.TrimPrefix(token, candidate))
				break
			}
		}
		bound, valid := parsePluginVersion(raw)
		if !valid {
			return false
		}
		comparison := comparePluginVersions(v, bound)
		matched := map[string]bool{
			">=": comparison >= 0,
			"<=": comparison <= 0,
			">":  comparison > 0,
			"<":  comparison < 0,
			"=":  comparison == 0,
		}[op]
		if !matched {
			return false
		}
	}
	return true
}
