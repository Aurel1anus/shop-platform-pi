package order

import (
	"context"

	"github.com/LYP-leo/xzxg-shop/commerce/internal/domain"
)

type Repository interface {
	CancelOrder(ctx context.Context, accountID string, orderID string, reason string) (domain.Order, bool)
	ConfirmReceipt(ctx context.Context, accountID string, orderID string) (domain.Order, bool)
	CreateOrderFromCart(ctx context.Context, accountID string) ([]domain.Order, bool)
	GetOrder(ctx context.Context, accountID string, orderID string) (domain.Order, bool)
	ListAllOrdersPage(ctx context.Context, page int, pageSize int) ([]domain.Order, int)
	ListMerchantOrdersPage(ctx context.Context, merchantID string, page int, pageSize int) ([]domain.Order, int)
	ListUserOrdersPage(ctx context.Context, accountID string, page int, pageSize int) ([]domain.Order, int)
	PayOrder(ctx context.Context, accountID string, orderID string, method string) (domain.Order, domain.Payment, bool)
	UpdateOrderStatus(ctx context.Context, merchantID string, orderID string, status string) (domain.Order, bool)
}

type Service struct{ repo Repository }

func New(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) CancelOrder(ctx context.Context, accountID string, orderID string, reason string) (domain.Order, bool) {
	return s.repo.CancelOrder(ctx, accountID, orderID, reason)
}

func (s *Service) ConfirmReceipt(ctx context.Context, accountID string, orderID string) (domain.Order, bool) {
	return s.repo.ConfirmReceipt(ctx, accountID, orderID)
}

// CreateOrderFromCart delegates the entire atomic checkout to one repository transaction.
func (s *Service) CreateOrderFromCart(ctx context.Context, accountID string) ([]domain.Order, bool) {
	if accountID == "" {
		return nil, false
	}
	return s.repo.CreateOrderFromCart(ctx, accountID)
}

func (s *Service) GetOrder(ctx context.Context, accountID string, orderID string) (domain.Order, bool) {
	return s.repo.GetOrder(ctx, accountID, orderID)
}

func (s *Service) ListAllOrdersPage(ctx context.Context, page int, pageSize int) ([]domain.Order, int) {
	return s.repo.ListAllOrdersPage(ctx, page, pageSize)
}

func (s *Service) ListMerchantOrdersPage(ctx context.Context, merchantID string, page int, pageSize int) ([]domain.Order, int) {
	return s.repo.ListMerchantOrdersPage(ctx, merchantID, page, pageSize)
}

func (s *Service) ListUserOrdersPage(ctx context.Context, accountID string, page int, pageSize int) ([]domain.Order, int) {
	return s.repo.ListUserOrdersPage(ctx, accountID, page, pageSize)
}

func (s *Service) PayOrder(ctx context.Context, accountID string, orderID string, method string) (domain.Order, domain.Payment, bool) {
	return s.repo.PayOrder(ctx, accountID, orderID, method)
}

func (s *Service) UpdateOrderStatus(ctx context.Context, merchantID, orderID, status string) (domain.Order, bool) {
	if status != "pending_ship" && status != "shipped" && status != "completed" {
		return domain.Order{}, false
	}
	return s.repo.UpdateOrderStatus(ctx, merchantID, orderID, status)
}
