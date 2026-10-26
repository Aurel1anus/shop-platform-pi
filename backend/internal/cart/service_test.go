package cart

import (
	"context"
	"testing"
	"time"

	"github.com/LYP-leo/xzxg-shop/commerce/internal/domain"
)

type cartRepo struct {
	Repository
	added   int
	account string
}

func (r *cartRepo) GetProduct(context.Context, string) (domain.ProductDetail, bool) {
	return domain.ProductDetail{}, true
}
func (r *cartRepo) ListProductSKUs(context.Context, string) []domain.ProductSKU {
	return []domain.ProductSKU{{SkuID: "sku-owned"}}
}
func (r *cartRepo) AddCartItem(_ context.Context, account, _, _ string, quantity int) (domain.Cart, bool) {
	r.added = quantity
	r.account = account
	return domain.Cart{}, true
}
func (r *cartRepo) GetCart(context.Context, string) domain.Cart {
	return domain.Cart{Items: []domain.CartItem{
		{MerchantID: "m1", Price: "200.00", Quantity: 2, Selected: true},
		{MerchantID: "m2", Price: "999.00", Quantity: 1, Selected: false},
	}}
}
func (r *cartRepo) ListPromotions(context.Context, string) []domain.PromotionRule {
	return []domain.PromotionRule{{Scope: "merchant", MerchantID: "m1", Status: "active", Type: "full_reduction", ThresholdAmount: "300", DiscountAmount: "30", StartAt: time.Now().Add(-time.Hour), EndAt: time.Now().Add(time.Hour)}}
}
func (r *cartRepo) ListUserCoupons(context.Context, string) []domain.UserCoupon { return nil }

func TestAddCartValidatesSKUAndPreservesAccount(t *testing.T) {
	repo := &cartRepo{}
	svc := New(repo)
	if _, ok := svc.AddCartItem(context.Background(), "user-a", "product", "sku-other-product", 1); ok || repo.added != 0 {
		t.Fatal("a SKU belonging to another product must not be added")
	}
	if _, ok := svc.AddCartItem(context.Background(), "user-a", "product", "sku-owned", -1); ok {
		t.Fatal("negative quantities must be rejected")
	}
	if _, ok := svc.AddCartItem(context.Background(), "user-a", "product", "sku-owned", 0); !ok || repo.added != 1 || repo.account != "user-a" {
		t.Fatal("default quantity and authenticated account were not preserved")
	}
}

func TestDiscountOnlyUsesSelectedItemsAndMerchantScope(t *testing.T) {
	preview := New(&cartRepo{}).PreviewCartDiscount(context.Background(), "user-a")
	if preview.TotalAmount != "400.00" || preview.DiscountAmount != "30.00" || preview.PayAmount != "370.00" {
		t.Fatalf("unexpected preview: %+v", preview)
	}
}
