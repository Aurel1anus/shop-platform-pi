package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LYP-leo/xzxg-shop/commerce/internal/auth"
	"github.com/LYP-leo/xzxg-shop/commerce/internal/domain"
	"github.com/LYP-leo/xzxg-shop/commerce/internal/platform/configcenter"
)

type identityRepo struct{ auth.Repository }

func (*identityRepo) GetAccountByToken(_ context.Context, token string) (domain.Account, bool) {
	if token == "user-token" {
		return domain.Account{AccountID: "user-a", Role: domain.AccountRoleUser}, true
	}
	if token == "merchant-token" {
		return domain.Account{AccountID: "merchant-a", Role: domain.AccountRoleMerchant}, true
	}
	return domain.Account{}, false
}

func TestBusinessRoutesDoNotRequireAgentAndKeepRoleChecks(t *testing.T) {
	server := NewServer(Services{Auth: auth.New(&identityRepo{})}, configcenter.NewMemoryCenter(nil), slog.New(slog.NewTextHandler(io.Discard, nil)))
	handler := server.Routes()
	for _, tc := range []struct {
		method, path, token string
		status              int
	}{
		{http.MethodGet, "/api/v1/health", "", 200},
		{http.MethodGet, "/api/v1/cart", "", 401},
		{http.MethodGet, "/api/v1/admin/accounts", "user-token", 403},
		{http.MethodPost, "/api/v1/agent/sessions", "user-token", 503},
		{http.MethodPost, "/api/v1/eval/intent", "", 501},
	} {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, req)
			if res.Code != tc.status {
				t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
			}
		})
	}
}
