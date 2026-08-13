package routes

import (
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUserRoutesKeepYuexiangModelCatalogContract(t *testing.T) {
	source, err := os.ReadFile("user.go")
	require.NoError(t, err)

	route := regexp.MustCompile(`models\s*:=\s*authenticated\.Group\("/models"\)[\s\S]*?models\.GET\("/catalog",\s*h\.AvailableChannel\.ListModels\)`)
	require.Regexp(t, route, string(source), "GET /api/v1/models/catalog must remain registered to BillingService-backed ListModels")
}
