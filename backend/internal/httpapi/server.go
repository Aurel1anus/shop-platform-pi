package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/LYP-leo/xzxg-shop/commerce/internal/auth"
	"github.com/LYP-leo/xzxg-shop/commerce/internal/cart"
	"github.com/LYP-leo/xzxg-shop/commerce/internal/catalog"
	"github.com/LYP-leo/xzxg-shop/commerce/internal/files"
	"github.com/LYP-leo/xzxg-shop/commerce/internal/knowledge"
	"github.com/LYP-leo/xzxg-shop/commerce/internal/order"
	"github.com/LYP-leo/xzxg-shop/commerce/internal/platform/configcenter"
	"github.com/LYP-leo/xzxg-shop/commerce/internal/promotion"
	"github.com/LYP-leo/xzxg-shop/commerce/internal/review"
)

type Server struct {
	auth      *auth.Service
	catalog   *catalog.Service
	cart      *cart.Service
	order     *order.Service
	promotion *promotion.Service
	review    *review.Service
	knowledge *knowledge.Service
	files     *files.Service
	configs   configcenter.Center
	logger    *slog.Logger
}

type Services struct {
	Auth      *auth.Service
	Catalog   *catalog.Service
	Cart      *cart.Service
	Order     *order.Service
	Promotion *promotion.Service
	Review    *review.Service
	Knowledge *knowledge.Service
	Files     *files.Service
}

func NewServer(services Services, configs configcenter.Center, logger *slog.Logger) *Server {
	return &Server{auth: services.Auth, catalog: services.Catalog, cart: services.Cart, order: services.Order, promotion: services.Promotion, review: services.Review, knowledge: services.Knowledge, files: services.Files, configs: configs, logger: logger}
}

// Routes 集中声明所有 API 路由，并在最后套上跨域、鉴权、限流、日志和请求体限制中间件。
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", s.handleHealth)
	mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/register", s.handleRegister)
	mux.HandleFunc("POST /api/v1/auth/logout", s.handleLogout)
	mux.HandleFunc("GET /api/v1/auth/me", s.handleMe)
	mux.HandleFunc("GET /api/v1/account/profile", s.handleAccountProfile)
	mux.HandleFunc("PATCH /api/v1/account/profile", s.handleAccountProfile)
	mux.HandleFunc("PATCH /api/v1/account/contact", s.handleAccountContact)
	mux.HandleFunc("DELETE /api/v1/account", s.handleDeleteAccount)
	mux.HandleFunc("POST /api/v1/uploads/avatar", s.handleUploadAvatar)
	mux.Handle("GET /api/v1/uploads/avatar/", http.StripPrefix("/api/v1/uploads/avatar/", http.FileServer(http.Dir(avatarUploadRoot()))))
	mux.Handle("GET /api/v1/assets/ecommerce_agent_dataset/", http.StripPrefix("/api/v1/assets/ecommerce_agent_dataset/", http.FileServer(http.Dir(datasetAssetRoot()))))
	mux.HandleFunc("GET /api/v1/categories/tree", s.handleListCategories)
	mux.HandleFunc("GET /api/v1/merchants", s.handleListMerchants)
	mux.HandleFunc("GET /api/v1/products", s.handleListProducts)
	mux.HandleFunc("GET /api/v1/products/", s.handleProductAction)
	mux.HandleFunc("GET /api/v1/promotions", s.handleListPromotions)
	mux.HandleFunc("GET /api/v1/coupons/available", s.handleListCoupons)
	mux.HandleFunc("GET /api/v1/coupons/mine", s.handleListUserCoupons)
	mux.HandleFunc("POST /api/v1/coupons/", s.handleCouponAction)
	mux.HandleFunc("POST /api/v1/files", s.handleUploadFile)
	mux.HandleFunc("GET /api/v1/files/", s.handleGetFile)
	mux.HandleFunc("GET /api/v1/merchant/documents", s.handleListMerchantDocuments)
	mux.HandleFunc("POST /api/v1/merchant/documents", s.handleCreateMerchantDocument)
	mux.HandleFunc("POST /api/v1/merchant/unstructured-ingestions", s.handleCreateMerchantUnstructuredIngestion)
	mux.HandleFunc("GET /api/v1/merchant/orders", s.handleListMerchantOrders)
	mux.HandleFunc("PATCH /api/v1/merchant/orders/", s.handleUpdateMerchantOrder)
	mux.HandleFunc("GET /api/v1/merchant/promotions", s.handleListMerchantPromotions)
	mux.HandleFunc("POST /api/v1/merchant/promotions", s.handleCreateMerchantPromotion)
	mux.HandleFunc("PATCH /api/v1/merchant/promotions/", s.handleUpdateMerchantPromotion)
	mux.HandleFunc("GET /api/v1/merchant/reviews", s.handleListMerchantReviews)
	mux.HandleFunc("POST /api/v1/merchant/reviews/", s.handleMerchantReviewAction)
	mux.HandleFunc("POST /api/v1/merchant/products", s.handleCreateMerchantProduct)
	mux.HandleFunc("PATCH /api/v1/merchant/products/", s.handleUpdateMerchantProduct)
	mux.HandleFunc("DELETE /api/v1/merchant/products/", s.handleDeleteMerchantProduct)
	mux.HandleFunc("GET /api/v1/admin/accounts", s.handleListAdminAccounts)
	mux.HandleFunc("PATCH /api/v1/admin/accounts/", s.handleUpdateAdminAccount)
	mux.HandleFunc("GET /api/v1/admin/configs", s.handleListAdminConfigs)
	mux.HandleFunc("PATCH /api/v1/admin/configs/", s.handleUpdateAdminConfig)
	mux.HandleFunc("GET /api/v1/admin/documents", s.handleListAdminDocuments)
	mux.HandleFunc("POST /api/v1/admin/unstructured-ingestions", s.handleCreateAdminUnstructuredIngestion)
	mux.HandleFunc("GET /api/v1/admin/orders", s.handleListAdminOrders)
	mux.HandleFunc("PATCH /api/v1/admin/orders/", s.handleUpdateAdminOrder)
	mux.HandleFunc("GET /api/v1/admin/promotions", s.handleListAdminPromotions)
	mux.HandleFunc("POST /api/v1/admin/promotions", s.handleCreateAdminPromotion)
	mux.HandleFunc("PATCH /api/v1/admin/promotions/", s.handleUpdateAdminPromotion)
	mux.HandleFunc("GET /api/v1/admin/reviews", s.handleListAdminReviews)
	mux.HandleFunc("PATCH /api/v1/admin/reviews/", s.handleUpdateAdminReview)
	mux.HandleFunc("GET /api/v1/admin/merchants", s.handleListAdminMerchants)
	mux.HandleFunc("GET /api/v1/admin/products", s.handleListAdminProducts)
	mux.HandleFunc("PATCH /api/v1/admin/products/", s.handleUpdateAdminProduct)
	mux.HandleFunc("PATCH /api/v1/admin/merchants/", s.handleUpdateAdminMerchant)
	mux.HandleFunc("GET /api/v1/cart", s.handleGetCart)
	mux.HandleFunc("GET /api/v1/cart/discount-preview", s.handleCartDiscountPreview)
	mux.HandleFunc("POST /api/v1/cart/items", s.handleAddCartItem)
	mux.HandleFunc("PATCH /api/v1/cart/items/", s.handleCartItemAction)
	mux.HandleFunc("DELETE /api/v1/cart/items/", s.handleCartItemAction)
	mux.HandleFunc("GET /api/v1/orders", s.handleListUserOrders)
	mux.HandleFunc("POST /api/v1/orders:checkout", s.handleCheckout)
	mux.HandleFunc("GET /api/v1/orders/", s.handleUserOrderAction)
	mux.HandleFunc("POST /api/v1/orders/", s.handleUserOrderAction)
	mux.HandleFunc("/api/v1/agent/", s.handleAgentUnavailable)
	mux.HandleFunc("/api/v1/eval/", s.handleAgentUnavailable)
	mux.HandleFunc("/api/v1/admin/prompts", s.handleAgentUnavailable)
	mux.HandleFunc("/api/v1/admin/prompts/", s.handleAgentUnavailable)
	mux.HandleFunc("/api/v1/admin/agent/", s.handleAgentUnavailable)
	mux.HandleFunc("/api/v1/admin/evals", s.handleAgentUnavailable)
	mux.HandleFunc("/api/v1/admin/evals/", s.handleAgentUnavailable)
	mux.HandleFunc("/api/v1/admin/vector/", s.handleAgentUnavailable)
	mux.HandleFunc("/api/v1/search/image", s.handleAgentUnavailable)
	return s.withCORS(s.withRequestContext(s.withAccessLog(s.withBodyLimit(s.withAuth(s.withRateLimit(mux))))))
}

func (s *Server) handleAgentUnavailable(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "agent_not_configured", "Agent、语义检索和评测服务尚未接入")
}
