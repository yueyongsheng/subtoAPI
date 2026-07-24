package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/payment"
)

func TestZPayCreateWechatPayment(t *testing.T) {
	t.Parallel()

	const pkey = "test-zpay-key"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/mapi.php" {
			t.Errorf("path = %q, want /mapi.php", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		for key, want := range map[string]string{
			"pid":          "test-zpay-pid",
			"type":         "wxpay",
			"cid":          "21213",
			"out_trade_no": "sub2_zpay_1001",
			"notify_url":   "https://api.example.com/api/v1/payment/webhook/easypay",
			"return_url":   "https://api.example.com/payment/result",
			"name":         "API prepaid balance",
			"money":        "1.00",
			"clientip":     "203.0.113.10",
			"sign_type":    "MD5",
		} {
			if got := r.PostForm.Get(key); got != want {
				t.Fatalf("form[%s] = %q, want %q", key, got, want)
			}
		}
		params := make(map[string]string, len(r.PostForm))
		for key := range r.PostForm {
			params[key] = r.PostForm.Get(key)
		}
		if !easyPayVerifySign(params, pkey, r.PostForm.Get("sign")) {
			t.Fatal("ZPAY request signature is invalid")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":1,"msg":"ok","O_id":"zpay-order-1","payurl":"https://cashier.example.com/pay/1","qrcode":"wxp://pay/1"}`))
	}))
	defer server.Close()

	provider, err := NewEasyPay("zpay", map[string]string{
		"pid":         "test-zpay-pid",
		"pkey":        pkey,
		"apiBase":     server.URL,
		"notifyUrl":   "https://api.example.com/api/v1/payment/webhook/easypay",
		"returnUrl":   "https://api.example.com/payment/result",
		"cidWxpay":    "21213",
		"paymentMode": "qrcode",
	})
	if err != nil {
		t.Fatalf("NewEasyPay: %v", err)
	}

	resp, err := provider.CreatePayment(context.Background(), payment.CreatePaymentRequest{
		OrderID:     "sub2_zpay_1001",
		Amount:      "1.00",
		PaymentType: payment.TypeWxpay,
		Subject:     "API prepaid balance",
		ClientIP:    "203.0.113.10",
	})
	if err != nil {
		t.Fatalf("CreatePayment: %v", err)
	}
	if resp.TradeNo != "zpay-order-1" || resp.QRCode != "wxp://pay/1" || resp.PayURL != "https://cashier.example.com/pay/1" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestZPayNotificationVerification(t *testing.T) {
	t.Parallel()

	provider := newTestEasyPay(t, "https://zpayz.example")
	values := url.Values{
		"pid":          {"pid-1"},
		"name":         {"API prepaid balance"},
		"money":        {"1.00"},
		"out_trade_no": {"sub2_zpay_1002"},
		"trade_no":     {"zpay-trade-1002"},
		"trade_status": {"TRADE_SUCCESS"},
		"type":         {"wxpay"},
		"sign_type":    {"MD5"},
	}
	params := make(map[string]string, len(values))
	for key := range values {
		params[key] = values.Get(key)
	}
	values.Set("sign", easyPaySign(params, "pkey-1"))

	notification, err := provider.VerifyNotification(context.Background(), values.Encode(), nil)
	if err != nil {
		t.Fatalf("VerifyNotification: %v", err)
	}
	if notification.Status != payment.ProviderStatusSuccess || notification.OrderID != "sub2_zpay_1002" || notification.TradeNo != "zpay-trade-1002" || notification.Amount != 1 {
		t.Fatalf("unexpected notification: %+v", notification)
	}
	if notification.Metadata["pid"] != "pid-1" {
		t.Fatalf("pid metadata = %q", notification.Metadata["pid"])
	}

	values.Set("money", "99.00")
	if _, err := provider.VerifyNotification(context.Background(), values.Encode(), nil); err == nil {
		t.Fatal("tampered ZPAY amount should fail signature verification")
	}
}
