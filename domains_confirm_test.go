package cloud

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ConfirmTotal reaches the API, and its refusal comes back as a
// *PriceConfirmationError carrying the current price.
func TestDomainsPriceConfirmation(t *testing.T) {
	const base = "/v1/workspaces/ws/domains"
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = nil
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusConflict)
		code := "ERR_PRICE_CHANGED"
		if got["confirm_total"] == nil {
			code = "ERR_PRICE_CONFIRMATION_REQUIRED"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": false, "code": code, "message": "the price is ₦18,920", "total": 189200000,
		})
	}))
	defer server.Close()
	domains := NewClient(WithBaseURL(server.URL), WithToken("sk_ws_x")).Workspace("ws").Domains()
	ctx := context.Background()

	stale := int64(1)
	_, err := domains.Register(ctx, RegisterDomainInput{Name: "acme.com", ConfirmTotal: &stale})
	if got["confirm_total"] != float64(1) {
		t.Errorf("Register sent %v", got)
	}
	pce, ok := AsPriceConfirmation(err)
	if !ok || pce.Total != 189200000 || !pce.Changed || pce.Code != "ERR_PRICE_CHANGED" || pce.Error() != "the price is ₦18,920" {
		t.Fatalf("Register error = %#v", err)
	}
	if !IsConflict(err) {
		t.Error("a price confirmation is still a 409")
	}

	_, err = domains.RenewWithOptions(ctx, "d1", RenewDomainOptions{Years: 2})
	if got["years"] != float64(2) || got["confirm_total"] != nil {
		t.Errorf("RenewWithOptions sent %v", got)
	}
	if pce, ok := AsPriceConfirmation(err); !ok || pce.Changed || pce.Total != 189200000 {
		t.Fatalf("RenewWithOptions error = %#v", err)
	}
	total := int64(189200000)
	_, _ = domains.RenewWithOptions(ctx, "d1", RenewDomainOptions{ConfirmTotal: &total})
	if got["confirm_total"] != float64(189200000) {
		t.Errorf("RenewWithOptions sent %v", got)
	}
}

// Other 409s are left as they are.
func TestPriceConfirmationLeavesOtherConflicts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{"message": "acme.com is already registered"})
	}))
	defer server.Close()
	_, err := NewClient(WithBaseURL(server.URL)).Workspace("ws").Domains().Register(context.Background(), RegisterDomainInput{Name: "acme.com"})
	if _, ok := AsPriceConfirmation(err); ok || !IsConflict(err) {
		t.Errorf("err = %#v", err)
	}
}
