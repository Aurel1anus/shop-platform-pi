package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
)

func (s *Server) handleGetCart(w http.ResponseWriter, r *http.Request) {
	account, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, s.cart.GetCart(r.Context(), account.AccountID))
}

func (s *Server) handleCartDiscountPreview(w http.ResponseWriter, r *http.Request) {
	account, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, s.cart.PreviewCartDiscount(r.Context(), account.AccountID))
}

func (s *Server) handleAddCartItem(w http.ResponseWriter, r *http.Request) {
	account, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var request struct {
		ProductID string `json:"product_id"`
		SkuID     string `json:"sku_id"`
		Quantity  int    `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "请求 JSON 不合法")
		return
	}
	if request.ProductID == "" {
		writeError(w, http.StatusBadRequest, "empty_product_id", "商品 ID 不能为空")
		return
	}
	cart, ok := s.cart.AddCartItem(r.Context(), account.AccountID, request.ProductID, request.SkuID, request.Quantity)
	if !ok {
		writeError(w, http.StatusNotFound, "product_not_found", "商品不存在")
		return
	}
	writeJSON(w, http.StatusOK, cart)
}

func (s *Server) handleCartItemAction(w http.ResponseWriter, r *http.Request) {
	account, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	cartItemID := strings.TrimPrefix(r.URL.Path, "/api/v1/cart/items/")
	if cartItemID == "" {
		writeError(w, http.StatusNotFound, "not_found", "接口不存在")
		return
	}

	if r.Method == http.MethodDelete {
		cart, ok := s.cart.DeleteCartItem(r.Context(), account.AccountID, cartItemID)
		if !ok {
			writeError(w, http.StatusNotFound, "cart_item_not_found", "购物车项不存在")
			return
		}
		writeJSON(w, http.StatusOK, cart)
		return
	}

	var request struct {
		Quantity *int  `json:"quantity"`
		Selected *bool `json:"selected"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "请求 JSON 不合法")
		return
	}
	cart, ok := s.cart.UpdateCartItem(r.Context(), account.AccountID, cartItemID, request.Quantity, request.Selected)
	if !ok {
		writeError(w, http.StatusNotFound, "cart_item_not_found", "购物车项不存在")
		return
	}
	writeJSON(w, http.StatusOK, cart)
}
