package mysqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/LYP-leo/xzxg-shop/commerce/internal/domain"
)

func (s *MySQLStore) ListProductReviews(ctx context.Context, productID string) []domain.ProductReview {
	return s.listReviews(ctx, "WHERE r.product_id = ? AND r.status = 'visible'", productID)
}

func (s *MySQLStore) ListProductReviewsPage(ctx context.Context, productID string, page int, pageSize int) ([]domain.ProductReview, int) {
	page, pageSize = normalizePage(page, pageSize)
	total := s.countRowsWhere(ctx, "product_reviews", "product_id = ? AND status = 'visible'", productID)
	return s.listReviewsWithLimit(ctx, "WHERE r.product_id = ? AND r.status = 'visible'", pageSize, pageOffset(page, pageSize), productID), total
}

func (s *MySQLStore) CreateProductReview(ctx context.Context, accountID string, orderID string, orderItemID string, input domain.ProductReviewInput) (domain.ProductReview, bool) {
	if input.Rating < 1 || input.Rating > 5 || strings.TrimSpace(input.Content) == "" {
		return domain.ProductReview{}, false
	}
	var productID string
	var skuID string
	var status string
	// 评价必须绑定到当前用户的已完成订单项，避免未购买商品被随意刷评价。
	err := s.db.QueryRowContext(ctx, `
		SELECT oi.product_id, oi.sku_id, o.status
		FROM order_items oi
		JOIN orders o ON o.order_id = oi.order_id
		WHERE oi.order_item_id = ? AND oi.order_id = ? AND o.account_id = ?
	`, orderItemID, orderID, accountID).Scan(&productID, &skuID, &status)
	if err != nil || status != "completed" {
		return domain.ProductReview{}, false
	}
	// tags 以 JSON 存储，便于前端做标签化展示，也方便后续扩展为结构化评价维度。
	tagsJSON, err := json.Marshal(input.Tags)
	if err != nil {
		return domain.ProductReview{}, false
	}
	now := time.Now()
	review := domain.ProductReview{
		ReviewID:    nextID("rev"),
		OrderID:     orderID,
		OrderItemID: orderItemID,
		ProductID:   productID,
		SkuID:       skuID,
		AccountID:   accountID,
		Rating:      input.Rating,
		Content:     strings.TrimSpace(input.Content),
		Tags:        input.Tags,
		Status:      "visible",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO product_reviews (
			review_id, order_id, order_item_id, product_id, sku_id, account_id, rating, content, tags_json, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, review.ReviewID, review.OrderID, review.OrderItemID, review.ProductID, review.SkuID, review.AccountID,
		review.Rating, review.Content, string(tagsJSON), review.Status, review.CreatedAt, review.UpdatedAt)
	if err != nil {
		return domain.ProductReview{}, false
	}
	return review, true
}

func (s *MySQLStore) ListMerchantReviews(ctx context.Context, merchantID string) []domain.ProductReview {
	return s.listReviews(ctx, "JOIN products p ON p.product_id = r.product_id WHERE p.merchant_id = ?", merchantID)
}

func (s *MySQLStore) ListMerchantReviewsPage(ctx context.Context, merchantID string, page int, pageSize int) ([]domain.ProductReview, int) {
	page, pageSize = normalizePage(page, pageSize)
	total := 0
	_ = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM product_reviews r
		JOIN products p ON p.product_id = r.product_id
		WHERE p.merchant_id = ?
	`, merchantID).Scan(&total)
	return s.listReviewsWithLimit(ctx, "JOIN products p ON p.product_id = r.product_id WHERE p.merchant_id = ?", pageSize, pageOffset(page, pageSize), merchantID), total
}

func (s *MySQLStore) ReplyReview(ctx context.Context, merchantID string, reviewID string, reply string) (domain.ProductReview, bool) {
	reply = strings.TrimSpace(reply)
	if reply == "" {
		return domain.ProductReview{}, false
	}
	now := time.Now()
	// 回复时通过 products 关联校验商家归属，防止商家回复其他店铺的评价。
	result, err := s.db.ExecContext(ctx, `
		UPDATE product_reviews r
		JOIN products p ON p.product_id = r.product_id
		SET r.merchant_reply = ?, r.merchant_replied_at = ?, r.updated_at = ?
		WHERE r.review_id = ? AND p.merchant_id = ?
	`, truncateRunes(reply, 1000), now, now, reviewID, merchantID)
	if err != nil {
		return domain.ProductReview{}, false
	}
	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		return domain.ProductReview{}, false
	}
	reviews := s.listReviews(ctx, "JOIN products p ON p.product_id = r.product_id WHERE r.review_id = ? AND p.merchant_id = ?", reviewID, merchantID)
	if len(reviews) == 0 {
		return domain.ProductReview{}, false
	}
	return reviews[0], true
}

func (s *MySQLStore) ListAllReviews(ctx context.Context) []domain.ProductReview {
	return s.listReviews(ctx, "")
}

func (s *MySQLStore) ListAllReviewsPage(ctx context.Context, page int, pageSize int) ([]domain.ProductReview, int) {
	page, pageSize = normalizePage(page, pageSize)
	total := s.countRows(ctx, "product_reviews")
	return s.listReviewsWithLimit(ctx, "", pageSize, pageOffset(page, pageSize)), total
}

func (s *MySQLStore) UpdateReviewStatus(ctx context.Context, reviewID string, status string) (domain.ProductReview, bool) {
	now := time.Now()
	result, err := s.db.ExecContext(ctx, `UPDATE product_reviews SET status = ?, updated_at = ? WHERE review_id = ?`, status, now, reviewID)
	if err != nil {
		return domain.ProductReview{}, false
	}
	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		return domain.ProductReview{}, false
	}
	reviews := s.listReviews(ctx, "WHERE r.review_id = ?", reviewID)
	if len(reviews) == 0 {
		return domain.ProductReview{}, false
	}
	return reviews[0], true
}

func (s *MySQLStore) listReviews(ctx context.Context, where string, args ...any) []domain.ProductReview {
	return s.listReviewsQuery(ctx, where, "", args...)
}

func (s *MySQLStore) listReviewsWithLimit(ctx context.Context, where string, limit int, offset int, args ...any) []domain.ProductReview {
	return s.listReviewsQuery(ctx, where, " LIMIT ? OFFSET ?", append(args, limit, offset)...)
}

func (s *MySQLStore) listReviewsQuery(ctx context.Context, where string, suffix string, args ...any) []domain.ProductReview {
	// 评价查询统一走这个方法，保证用户昵称、标签 JSON、商家回复时间等字段解析一致。
	query := `
		SELECT r.review_id, r.order_id, r.order_item_id, r.product_id, r.sku_id, r.account_id,
			COALESCE(a.username, ''), r.rating, r.content, r.tags_json, r.status, COALESCE(r.merchant_reply, ''),
			r.merchant_replied_at, r.created_at, r.updated_at
		FROM product_reviews r
		LEFT JOIN accounts a ON a.account_id = r.account_id
	`
	if where != "" {
		query += " " + where
	}
	query += " ORDER BY r.created_at DESC, r.review_id DESC"
	query += suffix
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := make([]domain.ProductReview, 0)
	for rows.Next() {
		var item domain.ProductReview
		var tagsJSON string
		var repliedAt sql.NullTime
		if err := rows.Scan(
			&item.ReviewID, &item.OrderID, &item.OrderItemID, &item.ProductID, &item.SkuID, &item.AccountID,
			&item.Username, &item.Rating, &item.Content, &tagsJSON, &item.Status, &item.MerchantReply,
			&repliedAt, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil
		}
		_ = json.Unmarshal([]byte(tagsJSON), &item.Tags)
		if repliedAt.Valid {
			item.MerchantRepliedAt = repliedAt.Time
		}
		items = append(items, item)
	}
	return items
}
