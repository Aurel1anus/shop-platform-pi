package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
)

func (s *Server) handleListMerchantReviews(w http.ResponseWriter, r *http.Request) {
	account, ok := s.requireMerchant(w, r)
	if !ok {
		return
	}
	page, pageSize := readPagination(r)
	items, total := s.review.ListMerchantReviewsPage(r.Context(), account.MerchantID, page, pageSize)
	writeJSON(w, http.StatusOK, pagedPayload(items, page, pageSize, total))
}

func (s *Server) handleMerchantReviewAction(w http.ResponseWriter, r *http.Request) {
	account, ok := s.requireMerchant(w, r)
	if !ok {
		return
	}
	reviewID, action, ok := splitReviewAction(r.URL.Path, "/api/v1/merchant/reviews/")
	if !ok || action != "reply" {
		writeError(w, http.StatusNotFound, "not_found", "接口不存在")
		return
	}
	var request struct {
		Reply string `json:"reply"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "请求 JSON 不合法")
		return
	}
	review, ok := s.review.ReplyReview(r.Context(), account.MerchantID, reviewID, request.Reply)
	if !ok {
		writeError(w, http.StatusNotFound, "review_not_found", "评价不存在或不属于当前商家")
		return
	}
	writeJSON(w, http.StatusOK, review)
}

func (s *Server) handleListAdminReviews(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	page, pageSize := readPagination(r)
	items, total := s.review.ListAllReviewsPage(r.Context(), page, pageSize)
	writeJSON(w, http.StatusOK, pagedPayload(items, page, pageSize, total))
}

func (s *Server) handleUpdateAdminReview(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	reviewID := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/reviews/")
	var request struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "请求 JSON 不合法")
		return
	}
	if request.Status != "visible" && request.Status != "hidden" && request.Status != "deleted" {
		writeError(w, http.StatusBadRequest, "bad_status", "评价状态不合法")
		return
	}
	review, ok := s.review.UpdateReviewStatus(r.Context(), reviewID, request.Status)
	if !ok {
		writeError(w, http.StatusNotFound, "review_not_found", "评价不存在")
		return
	}
	writeJSON(w, http.StatusOK, review)
}
