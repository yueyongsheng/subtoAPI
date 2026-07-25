package service

import (
	"math"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/shopspring/decimal"
)

const defaultBalanceRechargeMultiplier = 1.0

type RechargePackage struct {
	PayAmount      float64 `json:"pay_amount"`
	CreditedAmount float64 `json:"credited_amount"`
	Badge          string  `json:"badge,omitempty"`
}

var promotionalRechargePackages = []RechargePackage{
	{PayAmount: 38, CreditedAmount: 1000},
	{PayAmount: 72, CreditedAmount: 2000, Badge: "recommended"},
	{PayAmount: 105, CreditedAmount: 3000},
	{PayAmount: 170, CreditedAmount: 5000, Badge: "best_value"},
}

func PromotionalRechargePackages() []RechargePackage {
	return append([]RechargePackage(nil), promotionalRechargePackages...)
}

func normalizeBalanceRechargeMultiplier(multiplier float64) float64 {
	if math.IsNaN(multiplier) || math.IsInf(multiplier, 0) || multiplier <= 0 {
		return defaultBalanceRechargeMultiplier
	}
	return multiplier
}

// normalizeSubscriptionUSDToCNYRate 将非法值归一为 0（换算关闭）。
// 与余额倍率不同，0 是合法状态：表示订阅保持 price 直付的存量行为。
func normalizeSubscriptionUSDToCNYRate(rate float64) float64 {
	if math.IsNaN(rate) || math.IsInf(rate, 0) || rate < 0 {
		return 0
	}
	return rate
}

func calculateCreditedBalance(paymentAmount, multiplier float64) float64 {
	for _, pkg := range promotionalRechargePackages {
		if decimal.NewFromFloat(paymentAmount).Equal(decimal.NewFromFloat(pkg.PayAmount)) {
			return pkg.CreditedAmount
		}
	}
	return decimal.NewFromFloat(paymentAmount).
		Mul(decimal.NewFromFloat(normalizeBalanceRechargeMultiplier(multiplier))).
		Round(2).
		InexactFloat64()
}

func calculateGatewayRefundAmount(orderAmount, payAmount, refundAmount float64, currency string) float64 {
	if orderAmount <= 0 || payAmount <= 0 || refundAmount <= 0 {
		return 0
	}
	fractionDigits := int32(payment.CurrencyMaxFractionDigits(currency))
	if math.Abs(refundAmount-orderAmount) <= paymentAmountToleranceForCurrency(currency) {
		return decimal.NewFromFloat(payAmount).Round(fractionDigits).InexactFloat64()
	}
	return decimal.NewFromFloat(payAmount).
		Mul(decimal.NewFromFloat(refundAmount)).
		Div(decimal.NewFromFloat(orderAmount)).
		Round(fractionDigits).
		InexactFloat64()
}
