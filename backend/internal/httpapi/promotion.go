package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/LYP-leo/xzxg-shop/commerce/internal/domain"
)

func (s *Server) handleListPromotions(w http.ResponseWriter, r *http.Request) {
	page, pageSize := readPagination(r)
	items, total := s.promotion.ListPromotionsPage(r.Context(), "", page, pageSize)
	writeJSON(w, http.StatusOK, pagedPayload(items, page, pageSize, total))
}

func (s *Server) handleListCoupons(w http.ResponseWriter, r *http.Request) {
	account, _ := accountFromContext(r.Context())
	page, pageSize := readPagination(r)
	items, total := s.promotion.ListCouponsPage(r.Context(), account.AccountID, page, pageSize)
	writeJSON(w, http.StatusOK, pagedPayload(items, page, pageSize, total))
}

func (s *Server) handleListUserCoupons(w http.ResponseWriter, r *http.Request) {
	account, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	page, pageSize := readPagination(r)
	items, total := s.promotion.ListUserCouponsPage(r.Context(), account.AccountID, page, pageSize)
	writeJSON(w, http.StatusOK, pagedPayload(items, page, pageSize, total))
}

func (s *Server) handleCouponAction(w http.ResponseWriter, r *http.Request) {
	account, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	couponID, action, ok := splitCouponAction(r.URL.Path)
	if !ok || action != "claim" {
		writeError(w, http.StatusNotFound, "not_found", "接口不存在")
		return
	}
	coupon, ok := s.promotion.ClaimCoupon(r.Context(), account.AccountID, couponID)
	if !ok {
		writeError(w, http.StatusConflict, "claim_failed", "优惠券不可领取，可能已领过、已过期或已领完")
		return
	}
	writeJSON(w, http.StatusOK, coupon)
}

func (s *Server) handleListMerchantPromotions(w http.ResponseWriter, r *http.Request) {
	account, ok := s.requireMerchant(w, r)
	if !ok {
		return
	}
	page, pageSize := readPagination(r)
	items, total := s.promotion.ListPromotionsPage(r.Context(), account.MerchantID, page, pageSize)
	writeJSON(w, http.StatusOK, pagedPayload(items, page, pageSize, total))
}

func (s *Server) handleCreateMerchantPromotion(w http.ResponseWriter, r *http.Request) {
	account, ok := s.requireMerchant(w, r)
	if !ok {
		return
	}
	var request domain.PromotionRuleInput
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "请求 JSON 不合法")
		return
	}
	request.Scope = "merchant"
	request.MerchantID = account.MerchantID
	promotion, err := s.promotion.CreatePromotion(r.Context(), request)
	if err != nil {
		writeError(w, http.StatusBadRequest, "create_promotion_failed", "创建促销失败")
		return
	}
	writeJSON(w, http.StatusOK, promotion)
}

func (s *Server) handleUpdateMerchantPromotion(w http.ResponseWriter, r *http.Request) {
	account, ok := s.requireMerchant(w, r)
	if !ok {
		return
	}
	promotionID := strings.TrimPrefix(r.URL.Path, "/api/v1/merchant/promotions/")
	var request struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "请求 JSON 不合法")
		return
	}
	if !allowedEnabledStatus(request.Status) {
		writeError(w, http.StatusBadRequest, "bad_status", "促销状态不合法")
		return
	}
	promotion, ok := s.promotion.UpdatePromotionStatus(r.Context(), promotionID, account.MerchantID, request.Status)
	if !ok {
		writeError(w, http.StatusNotFound, "promotion_not_found", "促销不存在或不属于当前商家")
		return
	}
	writeJSON(w, http.StatusOK, promotion)
}

func (s *Server) handleListAdminPromotions(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	page, pageSize := readPagination(r)
	items, total := s.promotion.ListPromotionsPage(r.Context(), "", page, pageSize)
	writeJSON(w, http.StatusOK, pagedPayload(items, page, pageSize, total))
}

func (s *Server) handleCreateAdminPromotion(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var request domain.PromotionRuleInput
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "请求 JSON 不合法")
		return
	}
	promotion, err := s.promotion.CreatePromotion(r.Context(), request)
	if err != nil {
		writeError(w, http.StatusBadRequest, "create_promotion_failed", "创建促销失败")
		return
	}
	writeJSON(w, http.StatusOK, promotion)
}

func (s *Server) handleUpdateAdminPromotion(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	promotionID := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/promotions/")
	var request struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "请求 JSON 不合法")
		return
	}
	if !allowedEnabledStatus(request.Status) {
		writeError(w, http.StatusBadRequest, "bad_status", "促销状态不合法")
		return
	}
	promotion, ok := s.promotion.UpdatePromotionStatus(r.Context(), promotionID, "", request.Status)
	if !ok {
		writeError(w, http.StatusNotFound, "promotion_not_found", "促销不存在")
		return
	}
	writeJSON(w, http.StatusOK, promotion)
}
