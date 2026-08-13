//go:build unit

package handler

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestCheckoutInfoResponseIncludesPromotionalRechargePackages(t *testing.T) {
	encoded, err := json.Marshal(checkoutInfoResponse{
		RechargePackages: service.PromotionalRechargePackages(),
	})
	require.NoError(t, err)

	var payload struct {
		RechargePackages []service.RechargePackage `json:"recharge_packages"`
	}
	require.NoError(t, json.Unmarshal(encoded, &payload))
	require.Equal(t, []service.RechargePackage{
		{PayAmount: 38, CreditedAmount: 1000},
		{PayAmount: 72, CreditedAmount: 2000, Badge: "recommended"},
		{PayAmount: 105, CreditedAmount: 3000},
		{PayAmount: 170, CreditedAmount: 5000, Badge: "best_value"},
	}, payload.RechargePackages)
}
