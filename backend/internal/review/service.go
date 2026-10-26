package review

import (
	"context"

	"github.com/LYP-leo/xzxg-shop/commerce/internal/domain"
)

type Repository interface {
	CreateProductReview(ctx context.Context, accountID string, orderID string, orderItemID string, input domain.ProductReviewInput) (domain.ProductReview, bool)
	ListAllReviewsPage(ctx context.Context, page int, pageSize int) ([]domain.ProductReview, int)
	ListMerchantReviewsPage(ctx context.Context, merchantID string, page int, pageSize int) ([]domain.ProductReview, int)
	ListProductReviewsPage(ctx context.Context, productID string, page int, pageSize int) ([]domain.ProductReview, int)
	ReplyReview(ctx context.Context, merchantID string, reviewID string, reply string) (domain.ProductReview, bool)
	UpdateReviewStatus(ctx context.Context, reviewID string, status string) (domain.ProductReview, bool)
}

type Service struct{ repo Repository }

func New(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) CreateProductReview(ctx context.Context, accountID string, orderID string, orderItemID string, input domain.ProductReviewInput) (domain.ProductReview, bool) {
	return s.repo.CreateProductReview(ctx, accountID, orderID, orderItemID, input)
}

func (s *Service) ListAllReviewsPage(ctx context.Context, page int, pageSize int) ([]domain.ProductReview, int) {
	return s.repo.ListAllReviewsPage(ctx, page, pageSize)
}

func (s *Service) ListMerchantReviewsPage(ctx context.Context, merchantID string, page int, pageSize int) ([]domain.ProductReview, int) {
	return s.repo.ListMerchantReviewsPage(ctx, merchantID, page, pageSize)
}

func (s *Service) ListProductReviewsPage(ctx context.Context, productID string, page int, pageSize int) ([]domain.ProductReview, int) {
	return s.repo.ListProductReviewsPage(ctx, productID, page, pageSize)
}

func (s *Service) ReplyReview(ctx context.Context, merchantID string, reviewID string, reply string) (domain.ProductReview, bool) {
	return s.repo.ReplyReview(ctx, merchantID, reviewID, reply)
}

func (s *Service) UpdateReviewStatus(ctx context.Context, reviewID string, status string) (domain.ProductReview, bool) {
	return s.repo.UpdateReviewStatus(ctx, reviewID, status)
}
