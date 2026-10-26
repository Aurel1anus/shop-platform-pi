package catalog

import (
	"context"
	"errors"
	"strings"

	"github.com/LYP-leo/xzxg-shop/commerce/internal/domain"
)

type Repository interface {
	CreateProduct(ctx context.Context, input domain.ProductUpsertInput) (domain.ProductDetail, error)
	GetProduct(ctx context.Context, productID string) (domain.ProductDetail, bool)
	ListAllMerchantsPage(ctx context.Context, page int, pageSize int) ([]domain.Merchant, int)
	ListAllProductsPage(ctx context.Context, page int, pageSize int) ([]domain.ProductCard, int)
	ListCategories(ctx context.Context) []domain.Category
	ListMerchantsPage(ctx context.Context, page int, pageSize int) ([]domain.Merchant, int)
	ListProductSKUs(ctx context.Context, productID string) []domain.ProductSKU
	ListProductsPage(ctx context.Context, keyword string, categoryID string, page int, pageSize int) ([]domain.ProductCard, int)
	UpdateMerchantStatus(ctx context.Context, merchantID string, status string) (domain.Merchant, bool)
	UpdateProduct(ctx context.Context, productID string, input domain.ProductUpsertInput) (domain.ProductDetail, bool)
	UpdateProductStatus(ctx context.Context, merchantID string, productID string, status string) (domain.ProductDetail, bool)
}

type Service struct{ repo Repository }

func New(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) CreateProduct(ctx context.Context, input domain.ProductUpsertInput) (domain.ProductDetail, error) {
	input = normalizeProductInput(input)
	if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.MerchantID) == "" {
		return domain.ProductDetail{}, errors.New("商品名称和商家不能为空")
	}
	return s.repo.CreateProduct(ctx, input)
}

func (s *Service) GetProduct(ctx context.Context, productID string) (domain.ProductDetail, bool) {
	return s.repo.GetProduct(ctx, productID)
}

func (s *Service) ListAllMerchantsPage(ctx context.Context, page int, pageSize int) ([]domain.Merchant, int) {
	return s.repo.ListAllMerchantsPage(ctx, page, pageSize)
}

func (s *Service) ListAllProductsPage(ctx context.Context, page int, pageSize int) ([]domain.ProductCard, int) {
	return s.repo.ListAllProductsPage(ctx, page, pageSize)
}

func (s *Service) ListCategories(ctx context.Context) []domain.Category {
	return s.repo.ListCategories(ctx)
}

func (s *Service) ListMerchantsPage(ctx context.Context, page int, pageSize int) ([]domain.Merchant, int) {
	return s.repo.ListMerchantsPage(ctx, page, pageSize)
}

func (s *Service) ListProductSKUs(ctx context.Context, productID string) []domain.ProductSKU {
	return s.repo.ListProductSKUs(ctx, productID)
}

func (s *Service) ListProductsPage(ctx context.Context, keyword string, categoryID string, page int, pageSize int) ([]domain.ProductCard, int) {
	return s.repo.ListProductsPage(ctx, keyword, categoryID, page, pageSize)
}

func (s *Service) UpdateMerchantStatus(ctx context.Context, merchantID string, status string) (domain.Merchant, bool) {
	return s.repo.UpdateMerchantStatus(ctx, merchantID, status)
}

func (s *Service) UpdateProduct(ctx context.Context, productID string, input domain.ProductUpsertInput) (domain.ProductDetail, bool) {
	if strings.TrimSpace(productID) == "" || strings.TrimSpace(input.Name) == "" {
		return domain.ProductDetail{}, false
	}
	return s.repo.UpdateProduct(ctx, productID, normalizeProductInput(input))
}

func (s *Service) UpdateProductStatus(ctx context.Context, merchantID string, productID string, status string) (domain.ProductDetail, bool) {
	return s.repo.UpdateProductStatus(ctx, merchantID, productID, status)
}

func normalizeProductInput(input domain.ProductUpsertInput) domain.ProductUpsertInput {
	if input.Brand == "" {
		input.Brand = "未设置"
	}
	if input.CategoryID == "" {
		input.CategoryID = "c_phone"
	}
	if input.ImageURL == "" {
		input.ImageURL = "https://images.unsplash.com/photo-1516321318423-f06f85e504b3?auto=format&fit=crop&w=640&q=80"
	}
	if input.Price == "" {
		input.Price = "0.00"
	}
	if input.MarketPrice == "" {
		input.MarketPrice = input.Price
	}
	if input.StockStatus == "" {
		input.StockStatus = "in_stock"
	}
	if input.Tags == nil {
		input.Tags = []string{}
	}
	if input.SellingPoints == nil {
		input.SellingPoints = []string{}
	}
	if input.RiskNotes == nil {
		input.RiskNotes = []string{}
	}
	return input
}
