package cart

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/LYP-leo/xzxg-shop/commerce/internal/domain"
)

type Repository interface {
	GetProduct(ctx context.Context, productID string) (domain.ProductDetail, bool)
	ListProductSKUs(ctx context.Context, productID string) []domain.ProductSKU
	AddCartItem(ctx context.Context, accountID string, productID string, skuID string, quantity int) (domain.Cart, bool)
	DeleteCartItem(ctx context.Context, accountID string, cartItemID string) (domain.Cart, bool)
	GetCart(ctx context.Context, accountID string) domain.Cart
	ListPromotions(ctx context.Context, merchantID string) []domain.PromotionRule
	ListUserCoupons(ctx context.Context, accountID string) []domain.UserCoupon
	UpdateCartItem(ctx context.Context, accountID string, cartItemID string, quantity *int, selected *bool) (domain.Cart, bool)
}

type Service struct{ repo Repository }

func New(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) AddCartItem(ctx context.Context, accountID, productID, skuID string, quantity int) (domain.Cart, bool) {
	accountID = strings.TrimSpace(accountID)
	productID = strings.TrimSpace(productID)
	skuID = strings.TrimSpace(skuID)
	if accountID == "" || productID == "" || quantity < 0 {
		return domain.Cart{}, false
	}
	if quantity == 0 {
		quantity = 1
	}
	if _, ok := s.repo.GetProduct(ctx, productID); !ok {
		return domain.Cart{}, false
	}
	if skuID != "" {
		found := false
		for _, sku := range s.repo.ListProductSKUs(ctx, productID) {
			if sku.SkuID == skuID {
				found = true
				break
			}
		}
		if !found {
			return domain.Cart{}, false
		}
	}
	return s.repo.AddCartItem(ctx, accountID, productID, skuID, quantity)
}

func (s *Service) DeleteCartItem(ctx context.Context, accountID string, cartItemID string) (domain.Cart, bool) {
	return s.repo.DeleteCartItem(ctx, accountID, cartItemID)
}

func (s *Service) GetCart(ctx context.Context, accountID string) domain.Cart {
	return s.repo.GetCart(ctx, accountID)
}

func (s *Service) PreviewCartDiscount(ctx context.Context, accountID string) domain.DiscountPreview {
	cart := s.repo.GetCart(ctx, accountID)
	total := 0.0
	merchantTotals := make(map[string]float64)
	// 优惠试算只基于购物车选中项，并同时维护整单金额和商家维度金额。
	for _, item := range cart.Items {
		if !item.Selected {
			continue
		}
		price, _ := parseAmount(item.Price)
		amount := price * float64(item.Quantity)
		total += amount
		merchantTotals[item.MerchantID] += amount
	}
	lines := make([]domain.DiscountLine, 0)
	discount := 0.0
	now := time.Now()
	for _, promotion := range s.repo.ListPromotions(ctx, "") {
		// 平台券按整单金额计算，商家券只按该商家的商品金额计算。
		if promotion.Status != "active" || now.Before(promotion.StartAt) || now.After(promotion.EndAt) {
			continue
		}
		base := total
		if promotion.Scope == "merchant" {
			base = merchantTotals[promotion.MerchantID]
		}
		lineAmount := discountAmountFor(base, promotion.Type, promotion.ThresholdAmount, promotion.DiscountAmount, promotion.DiscountRate)
		if lineAmount <= 0 {
			continue
		}
		discount += lineAmount
		lines = append(lines, domain.DiscountLine{Type: "promotion", ID: promotion.PromotionID, Name: promotion.Name, Amount: fmt.Sprintf("%.2f", lineAmount)})
	}
	for _, userCoupon := range s.repo.ListUserCoupons(ctx, accountID) {
		// 这里只做试算，不会把优惠券置为 used；真正核销应放在支付/下单链路里。
		if userCoupon.Status != "unused" || userCoupon.Coupon.Status != "active" || now.Before(userCoupon.Coupon.StartAt) || now.After(userCoupon.Coupon.EndAt) {
			continue
		}
		base := total
		if userCoupon.Coupon.Scope == "merchant" {
			base = merchantTotals[userCoupon.Coupon.MerchantID]
		}
		lineAmount := discountAmountFor(base, userCoupon.Coupon.Type, userCoupon.Coupon.ThresholdAmount, userCoupon.Coupon.DiscountAmount, "0")
		if lineAmount <= 0 {
			continue
		}
		discount += lineAmount
		lines = append(lines, domain.DiscountLine{Type: "coupon", ID: userCoupon.UserCouponID, Name: userCoupon.Coupon.Name, Amount: fmt.Sprintf("%.2f", lineAmount)})
	}
	if discount > total {
		discount = total
	}
	return domain.DiscountPreview{
		TotalAmount:    fmt.Sprintf("%.2f", total),
		DiscountAmount: fmt.Sprintf("%.2f", discount),
		PayAmount:      fmt.Sprintf("%.2f", total-discount),
		Lines:          lines,
	}
}

func (s *Service) UpdateCartItem(ctx context.Context, accountID, cartItemID string, quantity *int, selected *bool) (domain.Cart, bool) {
	if accountID == "" || cartItemID == "" || (quantity != nil && *quantity <= 0) {
		return domain.Cart{}, false
	}
	return s.repo.UpdateCartItem(ctx, accountID, cartItemID, quantity, selected)
}

func discountAmountFor(base float64, discountType string, threshold string, fixedAmount string, rate string) float64 {
	thresholdValue, _ := parseAmount(threshold)
	if thresholdValue > 0 && base < thresholdValue {
		return 0
	}
	switch discountType {
	case "percentage":
		rateValue, _ := parseAmount(rate)
		if rateValue <= 0 || rateValue >= 1 {
			return 0
		}
		return base * rateValue
	default:
		amount, _ := parseAmount(fixedAmount)
		if amount < 0 {
			return 0
		}
		if amount > base {
			return base
		}
		return amount
	}
}
func parseAmount(input string) (float64, error) { return strconv.ParseFloat(input, 64) }
