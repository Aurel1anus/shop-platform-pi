package mysqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/LYP-leo/xzxg-shop/commerce/internal/domain"
)

func (s *MySQLStore) CreateOrderFromCart(ctx context.Context, accountID string) ([]domain.Order, bool) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, false
	}
	defer tx.Rollback()

	selected, ok := s.lockSelectedCartItems(ctx, tx, accountID)
	if !ok || len(selected) == 0 {
		return nil, false
	}
	for _, item := range selected {
		if !s.reserveOrderStock(ctx, tx, item.ProductID, item.SkuID, item.Quantity) {
			return nil, false
		}
	}

	// 一个购物车可能包含多个商家的商品，按商家拆单，便于后续商家侧发货和售后。
	byMerchant := make(map[string][]domain.CartItem)
	for _, item := range selected {
		byMerchant[item.MerchantID] = append(byMerchant[item.MerchantID], item)
	}

	orders := make([]domain.Order, 0, len(byMerchant))
	for merchantID, items := range byMerchant {
		orderID := nextID("ord")
		total := 0.0
		merchantName := ""
		for _, item := range items {
			price, _ := parseAmount(item.Price)
			total += price * float64(item.Quantity)
			merchantName = item.MerchantName
		}
		amount := fmt.Sprintf("%.2f", total)
		createdAt := time.Now()
		orderNo := nextOrderNo(createdAt)
		deadline := createdAt.Add(30 * time.Minute)
		// 订单初始状态固定为待支付，同时写入支付截止时间，超时任务会自动关单。
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO orders (
				order_id, order_no, account_id, merchant_id, status, total_amount, discount_amount, pay_amount,
				payment_deadline_at, created_at, updated_at
			)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, orderID, orderNo, accountID, merchantID, "pending_payment", amount, "0.00", amount, deadline, createdAt, createdAt); err != nil {
			return nil, false
		}
		// 当前支付仍是内部模拟支付；保留 payments 表是为了让真实支付网关后续能无缝接入。
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO payments (payment_id, order_id, account_id, amount, status, method, expires_at, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, nextID("pay"), orderID, accountID, amount, "pending", "mock", deadline, createdAt, createdAt); err != nil {
			return nil, false
		}
		orderItems := make([]domain.OrderItem, 0, len(items))
		for _, item := range items {
			orderItem := domain.OrderItem{
				OrderItemID:  nextID("ord_item"),
				ProductID:    item.ProductID,
				SkuID:        item.SkuID,
				Name:         item.Name,
				ImageURL:     item.ImageURL,
				Price:        item.Price,
				Quantity:     item.Quantity,
				MerchantID:   item.MerchantID,
				MerchantName: item.MerchantName,
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO order_items (order_item_id, order_id, product_id, sku_id, name, image_url, price, quantity, merchant_id, merchant_name, created_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, orderItem.OrderItemID, orderID, orderItem.ProductID, orderItem.SkuID, orderItem.Name, orderItem.ImageURL, orderItem.Price, orderItem.Quantity, orderItem.MerchantID, orderItem.MerchantName, createdAt); err != nil {
				return nil, false
			}
			orderItems = append(orderItems, orderItem)
		}
		orders = append(orders, domain.Order{
			OrderID:           orderID,
			OrderNo:           orderNo,
			AccountID:         accountID,
			MerchantID:        merchantID,
			MerchantName:      merchantName,
			Status:            "pending_payment",
			TotalAmount:       amount,
			DiscountAmount:    "0.00",
			PayAmount:         amount,
			PaymentDeadlineAt: deadline,
			Items:             orderItems,
			CreatedAt:         createdAt,
			UpdatedAt:         createdAt,
		})
	}
	// 订单创建成功后清空已结算项，避免用户重复提交同一批购物车商品。
	if _, err := tx.ExecContext(ctx, `DELETE FROM cart_items WHERE account_id = ? AND selected = TRUE`, accountID); err != nil {
		return nil, false
	}
	if err := tx.Commit(); err != nil {
		return nil, false
	}
	return orders, true
}

func (s *MySQLStore) lockSelectedCartItems(ctx context.Context, tx *sql.Tx, accountID string) ([]domain.CartItem, bool) {
	rows, err := tx.QueryContext(ctx, `
		SELECT
			ci.cart_item_id,
			ci.product_id,
			ci.sku_id,
			ci.quantity,
			ci.selected,
			p.name,
			p.image_url,
			COALESCE(ps.price, p.price) AS price,
			p.stock_status,
			p.merchant_id,
			m.name AS merchant_name
		FROM cart_items ci
		JOIN products p ON p.product_id = ci.product_id AND p.status = 'active'
		JOIN merchants m ON m.merchant_id = p.merchant_id
		LEFT JOIN product_skus ps ON ps.product_id = p.product_id AND ps.sku_id = ci.sku_id
		WHERE ci.account_id = ? AND ci.selected = TRUE
		ORDER BY ci.created_at, ci.cart_item_id
		FOR UPDATE
	`, accountID)
	if err != nil {
		return nil, false
	}
	defer rows.Close()
	items := make([]domain.CartItem, 0)
	for rows.Next() {
		var item domain.CartItem
		if err := rows.Scan(
			&item.CartItemID,
			&item.ProductID,
			&item.SkuID,
			&item.Quantity,
			&item.Selected,
			&item.Name,
			&item.ImageURL,
			&item.Price,
			&item.StockStatus,
			&item.MerchantID,
			&item.MerchantName,
		); err != nil {
			return nil, false
		}
		if item.Quantity <= 0 {
			return nil, false
		}
		items = append(items, item)
	}
	return items, rows.Err() == nil
}

func (s *MySQLStore) reserveOrderStock(ctx context.Context, tx *sql.Tx, productID string, skuID string, quantity int) bool {
	if quantity <= 0 {
		return false
	}
	now := time.Now()
	if skuID != "" {
		result, err := tx.ExecContext(ctx, `
			UPDATE product_skus
			SET stock_quantity = stock_quantity - ?,
				stock_status = CASE WHEN stock_quantity - ? <= 0 THEN 'out_of_stock' ELSE stock_status END,
				updated_at = ?
			WHERE product_id = ? AND sku_id = ? AND stock_quantity >= ?
		`, quantity, quantity, now, productID, skuID, quantity)
		if err != nil {
			return false
		}
		affected, err := result.RowsAffected()
		if err != nil || affected == 0 {
			return false
		}
	}
	result, err := tx.ExecContext(ctx, `
		UPDATE products
		SET stock_quantity = stock_quantity - ?,
			stock_status = CASE WHEN stock_quantity - ? <= 0 THEN 'out_of_stock' ELSE stock_status END,
			updated_at = ?
		WHERE product_id = ? AND stock_quantity >= ?
	`, quantity, quantity, now, productID, quantity)
	if err != nil {
		return false
	}
	affected, err := result.RowsAffected()
	return err == nil && affected > 0
}

func (s *MySQLStore) ListUserOrders(ctx context.Context, accountID string) []domain.Order {
	// 查询订单前先收敛超时待支付订单，保证列表看到的是最新业务状态。
	s.ExpirePendingOrders(ctx)
	return s.listOrders(ctx, "WHERE o.account_id = ?", accountID)
}

func (s *MySQLStore) ListUserOrdersPage(ctx context.Context, accountID string, page int, pageSize int) ([]domain.Order, int) {
	s.ExpirePendingOrders(ctx)
	page, pageSize = normalizePage(page, pageSize)
	total := s.countRowsWhere(ctx, "orders", "account_id = ?", accountID)
	items := s.listOrdersWithLimit(ctx, "WHERE o.account_id = ?", pageSize, pageOffset(page, pageSize), accountID)
	return items, total
}

func (s *MySQLStore) ListMerchantOrders(ctx context.Context, merchantID string) []domain.Order {
	s.ExpirePendingOrders(ctx)
	return s.listOrders(ctx, "WHERE o.merchant_id = ?", merchantID)
}

func (s *MySQLStore) ListMerchantOrdersPage(ctx context.Context, merchantID string, page int, pageSize int) ([]domain.Order, int) {
	s.ExpirePendingOrders(ctx)
	page, pageSize = normalizePage(page, pageSize)
	total := s.countRowsWhere(ctx, "orders", "merchant_id = ?", merchantID)
	items := s.listOrdersWithLimit(ctx, "WHERE o.merchant_id = ?", pageSize, pageOffset(page, pageSize), merchantID)
	return items, total
}

func (s *MySQLStore) ListAllOrders(ctx context.Context) []domain.Order {
	s.ExpirePendingOrders(ctx)
	return s.listOrders(ctx, "")
}

func (s *MySQLStore) ListAllOrdersPage(ctx context.Context, page int, pageSize int) ([]domain.Order, int) {
	s.ExpirePendingOrders(ctx)
	page, pageSize = normalizePage(page, pageSize)
	total := s.countRows(ctx, "orders")
	items := s.listOrdersWithLimit(ctx, "", pageSize, pageOffset(page, pageSize))
	return items, total
}

func (s *MySQLStore) GetOrder(ctx context.Context, accountID string, orderID string) (domain.Order, bool) {
	s.ExpirePendingOrders(ctx)
	orders := s.listOrders(ctx, "WHERE o.order_id = ? AND o.account_id = ?", orderID, accountID)
	if len(orders) == 0 {
		return domain.Order{}, false
	}
	return orders[0], true
}

func (s *MySQLStore) PayOrder(ctx context.Context, accountID string, orderID string, method string) (domain.Order, domain.Payment, bool) {
	s.ExpirePendingOrders(ctx)
	method = strings.TrimSpace(method)
	if method == "" {
		method = "mock_balance"
	}
	now := time.Now()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Order{}, domain.Payment{}, false
	}
	defer tx.Rollback()

	var amount string
	var deadline sql.NullTime
	// 使用 FOR UPDATE 锁定订单行，避免同一订单被并发支付两次。
	err = tx.QueryRowContext(ctx, `
		SELECT pay_amount, payment_deadline_at
		FROM orders
		WHERE order_id = ? AND account_id = ? AND status = 'pending_payment'
		FOR UPDATE
	`, orderID, accountID).Scan(&amount, &deadline)
	if err != nil {
		return domain.Order{}, domain.Payment{}, false
	}
	// 锁内再次校验支付截止时间，处理查询和支付提交之间刚好过期的边界情况。
	if deadline.Valid && now.After(deadline.Time) {
		_, _ = tx.ExecContext(ctx, `
			UPDATE orders SET status = 'closed_timeout', closed_at = ?, cancel_reason = '支付超时自动关闭', updated_at = ?
			WHERE order_id = ?
		`, now, now, orderID)
		_, _ = tx.ExecContext(ctx, `
			UPDATE payments SET status = 'expired', updated_at = ?
			WHERE order_id = ? AND status = 'pending'
		`, now, orderID)
		_ = tx.Commit()
		return domain.Order{}, domain.Payment{}, false
	}
	// 支付成功后订单进入待发货，真实接入支付渠道时这里应由支付回调驱动。
	if _, err := tx.ExecContext(ctx, `
		UPDATE orders
		SET status = 'pending_ship', paid_at = ?, updated_at = ?
		WHERE order_id = ? AND account_id = ? AND status = 'pending_payment'
	`, now, now, orderID, accountID); err != nil {
		return domain.Order{}, domain.Payment{}, false
	}
	transactionNo := nextID("txn")
	if _, err := tx.ExecContext(ctx, `
		UPDATE payments
		SET status = 'success', method = ?, transaction_no = ?, paid_at = ?, updated_at = ?
		WHERE order_id = ? AND account_id = ? AND status = 'pending'
	`, method, transactionNo, now, now, orderID, accountID); err != nil {
		return domain.Order{}, domain.Payment{}, false
	}
	if err := tx.Commit(); err != nil {
		return domain.Order{}, domain.Payment{}, false
	}
	order, ok := s.GetOrder(ctx, accountID, orderID)
	if !ok {
		return domain.Order{}, domain.Payment{}, false
	}
	payment, _ := s.getLatestPayment(ctx, accountID, orderID)
	return order, payment, true
}

func (s *MySQLStore) CancelOrder(ctx context.Context, accountID string, orderID string, reason string) (domain.Order, bool) {
	s.ExpirePendingOrders(ctx)
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "用户取消"
	}
	now := time.Now()
	// 当前只允许用户取消待支付订单；已支付订单后续应走售后/退款流程。
	result, err := s.db.ExecContext(ctx, `
		UPDATE orders
		SET status = 'canceled', closed_at = ?, cancel_reason = ?, updated_at = ?
		WHERE order_id = ? AND account_id = ? AND status = 'pending_payment'
	`, now, truncateRunes(reason, 256), now, orderID, accountID)
	if err != nil {
		return domain.Order{}, false
	}
	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		return domain.Order{}, false
	}
	_, _ = s.db.ExecContext(ctx, `
		UPDATE payments SET status = 'failed', updated_at = ?
		WHERE order_id = ? AND account_id = ? AND status = 'pending'
	`, now, orderID, accountID)
	return s.GetOrder(ctx, accountID, orderID)
}

func (s *MySQLStore) ConfirmReceipt(ctx context.Context, accountID string, orderID string) (domain.Order, bool) {
	s.ExpirePendingOrders(ctx)
	now := time.Now()
	// 用户只能确认已发货订单，确认后进入 completed，之后才允许发布商品评价。
	result, err := s.db.ExecContext(ctx, `
		UPDATE orders
		SET status = 'completed', completed_at = ?, updated_at = ?
		WHERE order_id = ? AND account_id = ? AND status = 'shipped'
	`, now, now, orderID, accountID)
	if err != nil {
		return domain.Order{}, false
	}
	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		return domain.Order{}, false
	}
	return s.GetOrder(ctx, accountID, orderID)
}

func (s *MySQLStore) ExpirePendingOrders(ctx context.Context) int {
	now := time.Now()
	// 这是轻量的惰性过期机制：每次订单相关查询/操作都会顺手关闭超时待支付订单。
	result, err := s.db.ExecContext(ctx, `
		UPDATE orders
		SET status = 'closed_timeout', closed_at = ?, cancel_reason = '支付超时自动关闭', updated_at = ?
		WHERE status = 'pending_payment' AND payment_deadline_at IS NOT NULL AND payment_deadline_at < ?
	`, now, now, now)
	if err != nil {
		return 0
	}
	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		return 0
	}
	// 订单超时后同步把仍处于 pending 的支付单标记为 expired，保持两张表状态一致。
	_, _ = s.db.ExecContext(ctx, `
		UPDATE payments p
		JOIN orders o ON o.order_id = p.order_id
		SET p.status = 'expired', p.updated_at = ?
		WHERE p.status = 'pending' AND o.status = 'closed_timeout'
	`, now)
	return int(affected)
}

func (s *MySQLStore) UpdateOrderStatus(ctx context.Context, merchantID string, orderID string, status string) (domain.Order, bool) {
	s.ExpirePendingOrders(ctx)
	now := time.Now()
	// 商家侧目前支持待发货/已发货之间的流转；管理员可传空 merchantID 跨商家操作。
	sets := `status = ?, updated_at = ?`
	args := []any{status, now}
	if status == "completed" {
		sets += `, completed_at = ?`
		args = append(args, now)
	}
	query := `UPDATE orders SET ` + sets + ` WHERE order_id = ?`
	args = append(args, orderID)
	if merchantID != "" {
		query += ` AND merchant_id = ?`
		args = append(args, merchantID)
	}
	query += ` AND status IN ('pending_ship', 'shipped')`
	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return domain.Order{}, false
	}
	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		return domain.Order{}, false
	}
	orders := s.listOrders(ctx, "WHERE o.order_id = ?", orderID)
	if len(orders) == 0 {
		return domain.Order{}, false
	}
	return orders[0], true
}

func (s *MySQLStore) listOrders(ctx context.Context, where string, args ...any) []domain.Order {
	return s.listOrdersQuery(ctx, where, "", args...)
}

func (s *MySQLStore) listOrdersWithLimit(ctx context.Context, where string, limit int, offset int, args ...any) []domain.Order {
	return s.listOrdersQuery(ctx, where, " LIMIT ? OFFSET ?", append(args, limit, offset)...)
}

func (s *MySQLStore) listOrdersQuery(ctx context.Context, where string, suffix string, args ...any) []domain.Order {
	query := `
		SELECT
			o.order_id, o.order_no, o.account_id, o.merchant_id, m.name, o.status,
			o.total_amount, o.discount_amount, o.pay_amount, o.payment_deadline_at,
			o.paid_at, o.closed_at, o.completed_at, o.cancel_reason, o.created_at, o.updated_at
		FROM orders o
		JOIN merchants m ON m.merchant_id = o.merchant_id
	`
	if where != "" {
		query += " " + where
	}
	query += " ORDER BY o.created_at DESC, o.order_id DESC"
	query += suffix
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	orders := make([]domain.Order, 0)
	for rows.Next() {
		var order domain.Order
		var deadline sql.NullTime
		var paidAt sql.NullTime
		var closedAt sql.NullTime
		var completedAt sql.NullTime
		if err := rows.Scan(
			&order.OrderID,
			&order.OrderNo,
			&order.AccountID,
			&order.MerchantID,
			&order.MerchantName,
			&order.Status,
			&order.TotalAmount,
			&order.DiscountAmount,
			&order.PayAmount,
			&deadline,
			&paidAt,
			&closedAt,
			&completedAt,
			&order.CancelReason,
			&order.CreatedAt,
			&order.UpdatedAt,
		); err != nil {
			return nil
		}
		if deadline.Valid {
			order.PaymentDeadlineAt = deadline.Time
		}
		if paidAt.Valid {
			order.PaidAt = paidAt.Time
		}
		if closedAt.Valid {
			order.ClosedAt = closedAt.Time
		}
		if completedAt.Valid {
			order.CompletedAt = completedAt.Time
		}
		order.Items = s.listOrderItems(ctx, order.OrderID)
		orders = append(orders, order)
	}
	return orders
}

func (s *MySQLStore) getLatestPayment(ctx context.Context, accountID string, orderID string) (domain.Payment, bool) {
	var payment domain.Payment
	var paidAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT payment_id, order_id, account_id, amount, status, method, transaction_no, expires_at, paid_at, created_at, updated_at
		FROM payments
		WHERE account_id = ? AND order_id = ?
		ORDER BY created_at DESC, payment_id DESC
		LIMIT 1
	`, accountID, orderID).Scan(
		&payment.PaymentID,
		&payment.OrderID,
		&payment.AccountID,
		&payment.Amount,
		&payment.Status,
		&payment.Method,
		&payment.TransactionNo,
		&payment.ExpiresAt,
		&paidAt,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	)
	if err != nil {
		return domain.Payment{}, false
	}
	if paidAt.Valid {
		payment.PaidAt = paidAt.Time
	}
	return payment, true
}

func (s *MySQLStore) listOrderItems(ctx context.Context, orderID string) []domain.OrderItem {
	rows, err := s.db.QueryContext(ctx, `
		SELECT order_item_id, product_id, sku_id, name, image_url, price, quantity, merchant_id, merchant_name
		FROM order_items
		WHERE order_id = ?
		ORDER BY order_item_id
	`, orderID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	items := make([]domain.OrderItem, 0)
	for rows.Next() {
		var item domain.OrderItem
		if err := rows.Scan(&item.OrderItemID, &item.ProductID, &item.SkuID, &item.Name, &item.ImageURL, &item.Price, &item.Quantity, &item.MerchantID, &item.MerchantName); err != nil {
			return nil
		}
		items = append(items, item)
	}
	return items
}
