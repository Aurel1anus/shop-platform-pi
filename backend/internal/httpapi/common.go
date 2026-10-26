package httpapi

import (
	"context"
	"encoding/json"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/LYP-leo/xzxg-shop/commerce/internal/domain"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowedOrigin := s.allowedCORSOrigin(r.Context(), origin)
		if allowedOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) allowedCORSOrigin(ctx context.Context, origin string) string {
	allowed := strings.TrimSpace(s.configs.GetMap(ctx)["http.cors.allowed_origins"])
	if allowed == "" {
		allowed = "*"
	}
	if allowed == "*" {
		return "*"
	}
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return ""
	}
	for _, item := range strings.Split(allowed, ",") {
		if strings.TrimSpace(item) == origin {
			return origin
		}
	}
	return ""
}

func (s *Server) accountFromRequest(r *http.Request) (domain.Account, bool) {
	if account, ok := accountFromContext(r.Context()); ok {
		return account, true
	}
	token := bearerToken(r)
	if token == "" {
		return domain.Account{}, false
	}
	return s.auth.GetAccountByToken(r.Context(), token)
}

func bearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	if token == "" || token == header {
		return ""
	}
	return token
}

func (s *Server) requireUser(w http.ResponseWriter, r *http.Request) (domain.Account, bool) {
	account, ok := accountFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "请先登录")
		return domain.Account{}, false
	}
	if account.Role != domain.AccountRoleUser {
		writeError(w, http.StatusForbidden, "forbidden", "当前账号不是用户端账号")
		return domain.Account{}, false
	}
	return account, true
}

func (s *Server) requireMerchant(w http.ResponseWriter, r *http.Request) (domain.Account, bool) {
	account, ok := accountFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "请先登录")
		return domain.Account{}, false
	}
	if account.Role != domain.AccountRoleMerchant || account.MerchantID == "" {
		writeError(w, http.StatusForbidden, "forbidden", "当前账号不是商家")
		return domain.Account{}, false
	}
	return account, true
}

func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request) (domain.Account, bool) {
	account, ok := accountFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "请先登录")
		return domain.Account{}, false
	}
	if account.Role != domain.AccountRoleAdmin {
		writeError(w, http.StatusForbidden, "forbidden", "当前账号不是管理员")
		return domain.Account{}, false
	}
	return account, true
}

func datasetAssetRoot() string {
	if root := os.Getenv("ECOMMERCE_DATASET_ROOT"); root != "" {
		return root
	}
	return filepath.Clean(filepath.Join("..", "quality", "data", "ecommerce_agent_dataset"))
}

func allowedOrderStatus(status string) bool {
	switch status {
	case "pending_ship", "shipped", "completed", "canceled", "refund_requested", "refunded":
		return true
	default:
		return false
	}
}

func allowedEnabledStatus(status string) bool {
	return status == domain.ProductStatusActive || status == domain.ProductStatusInactive
}

func allowedAccountStatus(status string) bool {
	switch status {
	case domain.AccountStatusActive, domain.AccountStatusInactive, domain.AccountStatusRisk:
		return true
	default:
		return false
	}
}

func allowedProductStatus(status string) bool {
	switch status {
	case domain.ProductStatusActive, domain.ProductStatusInactive, domain.ProductStatusDeleted, domain.ProductStatusRisk:
		return true
	default:
		return false
	}
}

func allowedMerchantStatus(status string) bool {
	switch status {
	case domain.MerchantStatusActive, domain.MerchantStatusInactive, domain.MerchantStatusRisk:
		return true
	default:
		return false
	}
}

func readPositiveIntQuery(r *http.Request, key string, fallback int, min int, max int) int {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	if parsed < min {
		return min
	}
	if parsed > max {
		return max
	}
	return parsed
}

func avatarUploadRoot() string {
	if root := strings.TrimSpace(os.Getenv("AVATAR_UPLOAD_ROOT")); root != "" {
		return root
	}
	return filepath.Join(".", "uploads", "avatar")
}

func int64FromMap(values map[string]string, key string, fallback int64) int64 {
	value := strings.TrimSpace(values[key])
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func fileExtension(filename string, mimeType string) string {
	if ext := strings.ToLower(filepath.Ext(filename)); ext != "" && len(ext) <= 12 {
		return ext
	}
	exts, err := mime.ExtensionsByType(mimeType)
	if err == nil && len(exts) > 0 {
		return exts[0]
	}
	return ".bin"
}

func readPagination(r *http.Request) (int, int) {
	page := readPositiveIntQuery(r, "page", 1, 1, 100000)
	pageSize := readPositiveIntQuery(r, "page_size", 10, 1, 100)
	return page, pageSize
}

func pagedPayload[T any](items []T, page int, pageSize int, total int) map[string]any {
	return map[string]any{
		"items":     items,
		"page":      page,
		"page_size": pageSize,
		"total":     total,
	}
}

func paginateList[T any](items []T, page int, pageSize int) ([]T, int) {
	total := len(items)
	start := (page - 1) * pageSize
	if start >= total {
		return []T{}, total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return items[start:end], total
}

func splitOrderAction(path string) (string, string, bool) {
	rest := strings.TrimPrefix(path, "/api/v1/orders/")
	if rest == "" {
		return "", "", false
	}
	parts := strings.Split(rest, ":")
	if len(parts) == 1 {
		return parts[0], "", false
	}
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		return parts[0], parts[1], true
	}
	return "", "", false
}

func splitOrderItemReviewPath(path string) (string, string, bool) {
	rest := strings.TrimPrefix(path, "/api/v1/orders/")
	parts := strings.Split(rest, "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] != "items" {
		return "", "", false
	}
	itemParts := strings.Split(parts[2], ":")
	if len(itemParts) != 2 || itemParts[0] == "" || itemParts[1] != "review" {
		return "", "", false
	}
	return parts[0], itemParts[0], true
}

func truncateText(input string, limit int) string {
	runes := []rune(strings.TrimSpace(input))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit])
}

func splitCouponAction(path string) (string, string, bool) {
	rest := strings.TrimPrefix(path, "/api/v1/coupons/")
	parts := strings.Split(rest, ":")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func splitReviewAction(path string, prefix string) (string, string, bool) {
	rest := strings.TrimPrefix(path, prefix)
	parts := strings.Split(rest, ":")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func splitProductAction(path string) (string, string, bool) {
	rest := strings.TrimPrefix(path, "/api/v1/products/")
	parts := strings.Split(rest, "/")
	if len(parts) == 1 && parts[0] != "" {
		return parts[0], "", true
	}
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		return parts[0], parts[1], true
	}
	return "", "", false
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code string, message string) {
	requestID := w.Header().Get("X-Request-ID")
	writeJSON(w, status, map[string]string{"code": code, "message": message, "request_id": requestID})
}
