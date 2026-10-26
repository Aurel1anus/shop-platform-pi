package mysqlstore

import (
	"context"
	"strings"
	"time"

	"github.com/LYP-leo/xzxg-shop/commerce/internal/domain"
)

func (s *MySQLStore) GetCart(ctx context.Context, accountID string) domain.Cart {
	return s.loadCart(ctx, accountID)
}

func (s *MySQLStore) AddCartItem(ctx context.Context, accountID string, productID string, skuID string, quantity int) (domain.Cart, bool) {
	if quantity <= 0 {
		quantity = 1
	}
	if _, ok := s.GetProduct(ctx, productID); !ok {
		return domain.Cart{}, false
	}
	// 前端可以只传商品 ID；未指定 SKU 时默认选择该商品的第一个 SKU。
	if skuID == "" {
		skuID = s.firstSKU(ctx, productID)
	}
	now := time.Now()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO cart_items (cart_item_id, account_id, product_id, sku_id, quantity, selected, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, TRUE, ?, ?)
		ON DUPLICATE KEY UPDATE
			quantity = quantity + VALUES(quantity),
			selected = TRUE,
			updated_at = VALUES(updated_at)
	`, nextID("cart"), accountID, productID, skuID, quantity, now, now)
	if err != nil {
		return domain.Cart{}, false
	}
	return s.loadCart(ctx, accountID), true
}

func (s *MySQLStore) UpdateCartItem(ctx context.Context, accountID string, cartItemID string, quantity *int, selected *bool) (domain.Cart, bool) {
	sets := make([]string, 0, 3)
	args := make([]any, 0, 4)
	if quantity != nil && *quantity > 0 {
		sets = append(sets, "quantity = ?")
		args = append(args, *quantity)
	}
	if selected != nil {
		sets = append(sets, "selected = ?")
		args = append(args, *selected)
	}
	if len(sets) == 0 {
		return s.loadCart(ctx, accountID), true
	}
	sets = append(sets, "updated_at = ?")
	args = append(args, time.Now(), cartItemID, accountID)
	result, err := s.db.ExecContext(ctx, `UPDATE cart_items SET `+strings.Join(sets, ", ")+` WHERE cart_item_id = ? AND account_id = ?`, args...)
	if err != nil {
		return domain.Cart{}, false
	}
	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		return domain.Cart{}, false
	}
	return s.loadCart(ctx, accountID), true
}

func (s *MySQLStore) DeleteCartItem(ctx context.Context, accountID string, cartItemID string) (domain.Cart, bool) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM cart_items WHERE cart_item_id = ? AND account_id = ?`, cartItemID, accountID)
	if err != nil {
		return domain.Cart{}, false
	}
	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		return domain.Cart{}, false
	}
	return s.loadCart(ctx, accountID), true
}

func (s *MySQLStore) loadCart(ctx context.Context, accountID string) domain.Cart {
	rows, err := s.db.QueryContext(ctx, `
		SELECT c.cart_item_id, c.product_id, c.sku_id, p.name, p.image_url,
			COALESCE(ps.price, p.price), c.quantity, c.selected,
			COALESCE(ps.stock_status, p.stock_status), p.merchant_id, m.name
		FROM cart_items c
		JOIN products p ON p.product_id = c.product_id
		JOIN merchants m ON m.merchant_id = p.merchant_id
		LEFT JOIN product_skus ps ON ps.sku_id = c.sku_id
		WHERE c.account_id = ?
		ORDER BY c.created_at, c.cart_item_id
	`, accountID)
	if err != nil {
		return domain.Cart{}
	}
	defer rows.Close()

	items := make([]domain.CartItem, 0)
	for rows.Next() {
		var item domain.CartItem
		if err := rows.Scan(
			&item.CartItemID,
			&item.ProductID,
			&item.SkuID,
			&item.Name,
			&item.ImageURL,
			&item.Price,
			&item.Quantity,
			&item.Selected,
			&item.StockStatus,
			&item.MerchantID,
			&item.MerchantName,
		); err != nil {
			return domain.Cart{}
		}
		items = append(items, item)
	}
	return buildCart(items)
}
