package promotion

import (
	"context"

	"github.com/LYP-leo/xzxg-shop/commerce/internal/domain"
)

type Repository interface {
	ClaimCoupon(ctx context.Context, accountID string, couponID string) (domain.UserCoupon, bool)
	CreatePromotion(ctx context.Context, input domain.PromotionRuleInput) (domain.PromotionRule, error)
	ListCouponsPage(ctx context.Context, accountID string, page int, pageSize int) ([]domain.Coupon, int)
	ListPromotionsPage(ctx context.Context, merchantID string, page int, pageSize int) ([]domain.PromotionRule, int)
	ListUserCouponsPage(ctx context.Context, accountID string, page int, pageSize int) ([]domain.UserCoupon, int)
	UpdatePromotionStatus(ctx context.Context, promotionID string, merchantID string, status string) (domain.PromotionRule, bool)
}

type Service struct{ repo Repository }

func New(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) ClaimCoupon(ctx context.Context, accountID string, couponID string) (domain.UserCoupon, bool) {
	return s.repo.ClaimCoupon(ctx, accountID, couponID)
}

func (s *Service) CreatePromotion(ctx context.Context, input domain.PromotionRuleInput) (domain.PromotionRule, error) {
	return s.repo.CreatePromotion(ctx, input)
}

func (s *Service) ListCouponsPage(ctx context.Context, accountID string, page int, pageSize int) ([]domain.Coupon, int) {
	return s.repo.ListCouponsPage(ctx, accountID, page, pageSize)
}

func (s *Service) ListPromotionsPage(ctx context.Context, merchantID string, page int, pageSize int) ([]domain.PromotionRule, int) {
	return s.repo.ListPromotionsPage(ctx, merchantID, page, pageSize)
}

func (s *Service) ListUserCouponsPage(ctx context.Context, accountID string, page int, pageSize int) ([]domain.UserCoupon, int) {
	return s.repo.ListUserCouponsPage(ctx, accountID, page, pageSize)
}

func (s *Service) UpdatePromotionStatus(ctx context.Context, promotionID string, merchantID string, status string) (domain.PromotionRule, bool) {
	return s.repo.UpdatePromotionStatus(ctx, promotionID, merchantID, status)
}
