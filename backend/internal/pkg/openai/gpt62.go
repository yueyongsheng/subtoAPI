package openai

import (
	_ "embed"
	"strings"
)

// GPT62ModelID recognizes only the two published IDs and existing local
// effort/compact suffixes. Invalid effort levels remain identifiable so they
// can be rejected before a request reaches the upstream.
func GPT62ModelID(model string) string {
	canonical := CanonicalizeOpenAIModelAliasSpelling(model)
	for _, id := range []string{"gpt-6.2-astra", "gpt-6.2-sol"} {
		if canonical == id {
			return id
		}
		if suffix, ok := strings.CutPrefix(canonical, id+"-"); ok {
			switch suffix {
			case "none", "minimal", "low", "medium", "high", "xhigh", "max", "openai-compact":
				return id
			}
		}
	}
	return ""
}

func IsGPT62ModelSpelling(model string) bool { return GPT62ModelID(model) != "" }

// Official Codex descriptors from openai/codex commit
// bb6b4b63b9df5432f4f45c90072d7823764ced00, codex-rs/models-manager/models.json.
// They are distinct from the public API's context/effort documentation.
//
//go:embed codex_gpt62_astra.json
var CodexGPT62AstraMetadata []byte

//go:embed codex_gpt62_sol.json
var CodexGPT62SolMetadata []byte

func GPT62CodexMetadata(model string) []byte {
	switch GPT62ModelID(model) {
	case "gpt-6.2-astra":
		return CodexGPT62AstraMetadata
	case "gpt-6.2-sol":
		return CodexGPT62SolMetadata
	default:
		return nil
	}
}
