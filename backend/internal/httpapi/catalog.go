package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/LYP-leo/xzxg-shop/commerce/internal/domain"
)

func (s *Server) handleListCategories(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"items": s.catalog.ListCategories(r.Context())})
}

func (s *Server) handleListMerchants(w http.ResponseWriter, r *http.Request) {
	page, pageSize := readPagination(r)
	items, total := s.catalog.ListMerchantsPage(r.Context(), page, pageSize)
	writeJSON(w, http.StatusOK, pagedPayload(items, page, pageSize, total))
}

func (s *Server) handleListProducts(w http.ResponseWriter, r *http.Request) {
	keyword := r.URL.Query().Get("keyword")
	categoryID := r.URL.Query().Get("category_id")
	page, pageSize := readPagination(r)
	items, total := s.catalog.ListProductsPage(r.Context(), keyword, categoryID, page, pageSize)
	writeJSON(w, http.StatusOK, pagedPayload(items, page, pageSize, total))
}

func (s *Server) handleProductAction(w http.ResponseWriter, r *http.Request) {
	productID, action, ok := splitProductAction(r.URL.Path)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "接口不存在")
		return
	}
	if action == "skus" {
		// SKU 与商品详情分开查询，前端可在详情页懒加载规格信息。
		page, pageSize := readPagination(r)
		items, total := paginateList(s.catalog.ListProductSKUs(r.Context(), productID), page, pageSize)
		writeJSON(w, http.StatusOK, pagedPayload(items, page, pageSize, total))
		return
	}
	if action == "reviews" {
		// 商品评价是公开信息，只返回 visible 状态的评价。
		page, pageSize := readPagination(r)
		items, total := s.review.ListProductReviewsPage(r.Context(), productID, page, pageSize)
		writeJSON(w, http.StatusOK, pagedPayload(items, page, pageSize, total))
		return
	}
	if action != "" {
		writeError(w, http.StatusNotFound, "not_found", "接口不存在")
		return
	}
	product, ok := s.catalog.GetProduct(r.Context(), productID)
	if !ok {
		writeError(w, http.StatusNotFound, "product_not_found", "商品不存在")
		return
	}
	writeJSON(w, http.StatusOK, product)
}

func (s *Server) handleCreateMerchantProduct(w http.ResponseWriter, r *http.Request) {
	account, ok := s.requireMerchant(w, r)
	if !ok {
		return
	}
	var request domain.ProductUpsertInput
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "请求 JSON 不合法")
		return
	}
	if strings.TrimSpace(request.Name) == "" {
		writeError(w, http.StatusBadRequest, "empty_name", "商品名称不能为空")
		return
	}
	request.MerchantID = account.MerchantID
	product, err := s.catalog.CreateProduct(r.Context(), request)
	if err != nil {
		s.logger.Error("create merchant product failed", "error", err, "merchant_id", account.MerchantID)
		writeError(w, http.StatusInternalServerError, "create_product_failed", "创建商品失败")
		return
	}
	writeJSON(w, http.StatusOK, product)
}

func (s *Server) handleUpdateMerchantProduct(w http.ResponseWriter, r *http.Request) {
	account, ok := s.requireMerchant(w, r)
	if !ok {
		return
	}
	productID := strings.TrimPrefix(r.URL.Path, "/api/v1/merchant/products/")
	if productID == "" {
		writeError(w, http.StatusNotFound, "not_found", "接口不存在")
		return
	}
	var request domain.ProductUpsertInput
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "请求 JSON 不合法")
		return
	}
	if strings.TrimSpace(request.Name) == "" {
		writeError(w, http.StatusBadRequest, "empty_name", "商品名称不能为空")
		return
	}
	request.MerchantID = account.MerchantID
	product, ok := s.catalog.UpdateProduct(r.Context(), productID, request)
	if !ok {
		writeError(w, http.StatusNotFound, "product_not_found", "商品不存在或不属于当前商家")
		return
	}
	writeJSON(w, http.StatusOK, product)
}

func (s *Server) handleDeleteMerchantProduct(w http.ResponseWriter, r *http.Request) {
	account, ok := s.requireMerchant(w, r)
	if !ok {
		return
	}
	productID := strings.TrimPrefix(r.URL.Path, "/api/v1/merchant/products/")
	if productID == "" {
		writeError(w, http.StatusNotFound, "not_found", "接口不存在")
		return
	}
	if _, ok := s.catalog.UpdateProductStatus(r.Context(), account.MerchantID, productID, "deleted"); !ok {
		writeError(w, http.StatusNotFound, "product_not_found", "商品不存在或不属于当前商家")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"product_id": productID, "status": "deleted"})
}

func (s *Server) handleListAdminProducts(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	page := readPositiveIntQuery(r, "page", 1, 1, 100000)
	pageSize := readPositiveIntQuery(r, "page_size", 10, 1, 100)
	items, total := s.catalog.ListAllProductsPage(r.Context(), page, pageSize)
	writeJSON(w, http.StatusOK, pagedPayload(items, page, pageSize, total))
}

func (s *Server) handleListAdminMerchants(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	page, pageSize := readPagination(r)
	items, total := s.catalog.ListAllMerchantsPage(r.Context(), page, pageSize)
	writeJSON(w, http.StatusOK, pagedPayload(items, page, pageSize, total))
}

func (s *Server) handleUpdateAdminProduct(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	productID := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/products/")
	var request struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "请求 JSON 不合法")
		return
	}
	if !allowedProductStatus(request.Status) {
		writeError(w, http.StatusBadRequest, "bad_status", "商品状态不合法")
		return
	}
	product, ok := s.catalog.UpdateProductStatus(r.Context(), "", productID, request.Status)
	if !ok {
		writeError(w, http.StatusNotFound, "product_not_found", "商品不存在")
		return
	}
	writeJSON(w, http.StatusOK, product)
}

func (s *Server) handleUpdateAdminMerchant(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	merchantID := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/merchants/")
	var request struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "请求 JSON 不合法")
		return
	}
	if !allowedMerchantStatus(request.Status) {
		writeError(w, http.StatusBadRequest, "bad_status", "商家状态不合法")
		return
	}
	merchant, ok := s.catalog.UpdateMerchantStatus(r.Context(), merchantID, request.Status)
	if !ok {
		writeError(w, http.StatusNotFound, "merchant_not_found", "商家不存在")
		return
	}
	writeJSON(w, http.StatusOK, merchant)
}
