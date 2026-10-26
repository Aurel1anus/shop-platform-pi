package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/LYP-leo/xzxg-shop/commerce/internal/domain"
)

func (s *Server) handleListMerchantOrders(w http.ResponseWriter, r *http.Request) {
	account, ok := s.requireMerchant(w, r)
	if !ok {
		return
	}
	page, pageSize := readPagination(r)
	items, total := s.order.ListMerchantOrdersPage(r.Context(), account.MerchantID, page, pageSize)
	writeJSON(w, http.StatusOK, pagedPayload(items, page, pageSize, total))
}

func (s *Server) handleUpdateMerchantOrder(w http.ResponseWriter, r *http.Request) {
	account, ok := s.requireMerchant(w, r)
	if !ok {
		return
	}
	orderID := strings.TrimPrefix(r.URL.Path, "/api/v1/merchant/orders/")
	var request struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "请求 JSON 不合法")
		return
	}
	if !allowedOrderStatus(request.Status) {
		writeError(w, http.StatusBadRequest, "bad_status", "订单状态不合法")
		return
	}
	order, ok := s.order.UpdateOrderStatus(r.Context(), account.MerchantID, orderID, request.Status)
	if !ok {
		writeError(w, http.StatusNotFound, "order_not_found", "订单不存在或不属于当前商家")
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (s *Server) handleListAdminOrders(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	page, pageSize := readPagination(r)
	items, total := s.order.ListAllOrdersPage(r.Context(), page, pageSize)
	writeJSON(w, http.StatusOK, pagedPayload(items, page, pageSize, total))
}

func (s *Server) handleUpdateAdminOrder(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	orderID := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/orders/")
	orderID = strings.Trim(orderID, "/")
	if orderID == "" {
		writeError(w, http.StatusBadRequest, "empty_order_id", "订单 ID 不能为空")
		return
	}
	var request struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "请求 JSON 不合法")
		return
	}
	if !allowedOrderStatus(request.Status) {
		writeError(w, http.StatusBadRequest, "bad_status", "订单状态不合法")
		return
	}
	order, ok := s.order.UpdateOrderStatus(r.Context(), "", orderID, request.Status)
	if !ok {
		writeError(w, http.StatusConflict, "order_status_update_failed", "订单不存在或当前状态不允许更新")
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (s *Server) handleListUserOrders(w http.ResponseWriter, r *http.Request) {
	account, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	page, pageSize := readPagination(r)
	items, total := s.order.ListUserOrdersPage(r.Context(), account.AccountID, page, pageSize)
	writeJSON(w, http.StatusOK, pagedPayload(items, page, pageSize, total))
}

func (s *Server) handleCheckout(w http.ResponseWriter, r *http.Request) {
	account, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	orders, ok := s.order.CreateOrderFromCart(r.Context(), account.AccountID)
	if !ok {
		writeError(w, http.StatusBadRequest, "empty_cart", "请选择购物车商品后再下单")
		return
	}
	// checkout 只创建待支付订单，不直接进入待发货；支付动作由 /orders/{id}:pay 完成。
	writeJSON(w, http.StatusOK, map[string]any{"items": orders})
}

func (s *Server) handleUserOrderAction(w http.ResponseWriter, r *http.Request) {
	account, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	if r.Method == http.MethodPost {
		if orderID, orderItemID, ok := splitOrderItemReviewPath(r.URL.Path); ok {
			// 评价强绑定已完成订单项，防止用户绕过订单直接刷评价。
			var request domain.ProductReviewInput
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				writeError(w, http.StatusBadRequest, "bad_request", "请求 JSON 不合法")
				return
			}
			review, ok := s.review.CreateProductReview(r.Context(), account.AccountID, orderID, orderItemID, request)
			if !ok {
				writeError(w, http.StatusConflict, "review_failed", "评价失败，只有已完成订单项可评价且不可重复评价")
				return
			}
			writeJSON(w, http.StatusOK, review)
			return
		}
	}
	orderID, action, hasAction := splitOrderAction(r.URL.Path)
	if orderID == "" {
		writeError(w, http.StatusNotFound, "not_found", "接口不存在")
		return
	}
	if r.Method == http.MethodGet && !hasAction {
		order, ok := s.order.GetOrder(r.Context(), account.AccountID, orderID)
		if !ok {
			writeError(w, http.StatusNotFound, "order_not_found", "订单不存在")
			return
		}
		writeJSON(w, http.StatusOK, order)
		return
	}
	if r.Method != http.MethodPost || !hasAction {
		writeError(w, http.StatusNotFound, "not_found", "接口不存在")
		return
	}
	switch action {
	case "pay":
		// 虚拟支付只推进本地订单状态，不接第三方支付网关。
		var request struct {
			Method string `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil && err.Error() != "EOF" {
			writeError(w, http.StatusBadRequest, "bad_request", "请求 JSON 不合法")
			return
		}
		order, payment, ok := s.order.PayOrder(r.Context(), account.AccountID, orderID, request.Method)
		if !ok {
			writeError(w, http.StatusConflict, "pay_failed", "订单不可支付，可能已支付、取消或超时关闭")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"order": order, "payment": payment})
	case "cancel":
		// MVP 只允许取消待支付订单；已支付订单后续走售后/退款流程。
		var request struct {
			Reason string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil && err.Error() != "EOF" {
			writeError(w, http.StatusBadRequest, "bad_request", "请求 JSON 不合法")
			return
		}
		order, ok := s.order.CancelOrder(r.Context(), account.AccountID, orderID, request.Reason)
		if !ok {
			writeError(w, http.StatusConflict, "cancel_failed", "订单不可取消，只有待支付订单可以取消")
			return
		}
		writeJSON(w, http.StatusOK, order)
	case "confirm-receipt":
		// 用户确认收货后订单进入 completed，随后才允许评价。
		order, ok := s.order.ConfirmReceipt(r.Context(), account.AccountID, orderID)
		if !ok {
			writeError(w, http.StatusConflict, "confirm_failed", "订单不可确认收货，只有已发货订单可以确认")
			return
		}
		writeJSON(w, http.StatusOK, order)
	default:
		writeError(w, http.StatusNotFound, "not_found", "接口不存在")
	}
}
