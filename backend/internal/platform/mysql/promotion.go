package mysqlstore

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/LYP-leo/xzxg-shop/commerce/internal/domain"
)

func (s *MySQLStore) ListPromotions(ctx context.Context, merchantID string) []domain.PromotionRule {
	return s.queryPromotions(ctx, merchantID, 0, 0)
}

func (s *MySQLStore) ListPromotionsPage(ctx context.Context, merchantID string, page int, pageSize int) ([]domain.PromotionRule, int) {
	page, pageSize = normalizePage(page, pageSize)
	total := s.countRows(ctx, "promotion_rules")
	if merchantID != "" {
		total = s.countRowsWhere(ctx, "promotion_rules", "merchant_id = ?", merchantID)
	}
	return s.queryPromotions(ctx, merchantID, pageSize, pageOffset(page, pageSize)), total
}

func (s *MySQLStore) queryPromotions(ctx context.Context, merchantID string, limit int, offset int) []domain.PromotionRule {
	query := `
		SELECT promotion_id, name, scope, merchant_id, product_id, category_id, type, threshold_amount,
			discount_amount, discount_rate, stackable, start_at, end_at, status, created_at, updated_at
		FROM promotion_rules
	`
	args := []any{}
	if merchantID != "" {
		query += ` WHERE merchant_id = ?`
		args = append(args, merchantID)
	}
	query += ` ORDER BY created_at DESC, promotion_id DESC`
	if limit > 0 {
		query += ` LIMIT ? OFFSET ?`
		args = append(args, limit, offset)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := make([]domain.PromotionRule, 0)
	for rows.Next() {
		var item domain.PromotionRule
		if err := rows.Scan(
			&item.PromotionID, &item.Name, &item.Scope, &item.MerchantID, &item.ProductID, &item.CategoryID, &item.Type,
			&item.ThresholdAmount, &item.DiscountAmount, &item.DiscountRate, &item.Stackable, &item.StartAt, &item.EndAt,
			&item.Status, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil
		}
		items = append(items, item)
	}
	return items
}

func (s *MySQLStore) CreatePromotion(ctx context.Context, input domain.PromotionRuleInput) (domain.PromotionRule, error) {
	now := time.Now()
	startAt := parseAPITime(input.StartAt, now)
	endAt := parseAPITime(input.EndAt, now.Add(30*24*time.Hour))
	if input.Scope == "" {
		input.Scope = "platform"
	}
	if input.Type == "" {
		input.Type = "full_reduction"
	}
	if input.Status == "" {
		input.Status = "active"
	}
	if input.ThresholdAmount == "" {
		input.ThresholdAmount = "0.00"
	}
	if input.DiscountAmount == "" {
		input.DiscountAmount = "0.00"
	}
	if input.DiscountRate == "" {
		input.DiscountRate = "0.0000"
	}
	item := domain.PromotionRule{
		PromotionID:     nextID("promo"),
		Name:            strings.TrimSpace(input.Name),
		Scope:           strings.TrimSpace(input.Scope),
		MerchantID:      strings.TrimSpace(input.MerchantID),
		ProductID:       strings.TrimSpace(input.ProductID),
		CategoryID:      strings.TrimSpace(input.CategoryID),
		Type:            strings.TrimSpace(input.Type),
		ThresholdAmount: input.ThresholdAmount,
		DiscountAmount:  input.DiscountAmount,
		DiscountRate:    input.DiscountRate,
		Stackable:       input.Stackable,
		StartAt:         startAt,
		EndAt:           endAt,
		Status:          input.Status,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if item.Name == "" {
		return domain.PromotionRule{}, errors.New("empty promotion name")
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO promotion_rules (
			promotion_id, name, scope, merchant_id, product_id, category_id, type, threshold_amount,
			discount_amount, discount_rate, stackable, start_at, end_at, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, item.PromotionID, item.Name, item.Scope, item.MerchantID, item.ProductID, item.CategoryID, item.Type,
		item.ThresholdAmount, item.DiscountAmount, item.DiscountRate, item.Stackable, item.StartAt, item.EndAt,
		item.Status, item.CreatedAt, item.UpdatedAt)
	if err != nil {
		return domain.PromotionRule{}, err
	}
	return item, nil
}

func (s *MySQLStore) UpdatePromotionStatus(ctx context.Context, promotionID string, merchantID string, status string) (domain.PromotionRule, bool) {
	args := []any{status, time.Now(), promotionID}
	query := `UPDATE promotion_rules SET status = ?, updated_at = ? WHERE promotion_id = ?`
	if merchantID != "" {
		query += ` AND merchant_id = ?`
		args = append(args, merchantID)
	}
	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return domain.PromotionRule{}, false
	}
	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		return domain.PromotionRule{}, false
	}
	items := s.ListPromotions(ctx, merchantID)
	for _, item := range items {
		if item.PromotionID == promotionID {
			return item, true
		}
	}
	return domain.PromotionRule{}, false
}

func (s *MySQLStore) ListCoupons(ctx context.Context, accountID string) []domain.Coupon {
	now := time.Now()
	rows, err := s.db.QueryContext(ctx, `
		SELECT coupon_id, name, scope, merchant_id, type, threshold_amount, discount_amount, total_count,
			claimed_count, per_user_limit, start_at, end_at, status, created_at, updated_at
		FROM coupons
		WHERE status = 'active' AND start_at <= ? AND end_at >= ?
		ORDER BY created_at DESC, coupon_id DESC
	`, now, now)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := make([]domain.Coupon, 0)
	for rows.Next() {
		var item domain.Coupon
		if err := rows.Scan(&item.CouponID, &item.Name, &item.Scope, &item.MerchantID, &item.Type, &item.ThresholdAmount,
			&item.DiscountAmount, &item.TotalCount, &item.ClaimedCount, &item.PerUserLimit, &item.StartAt, &item.EndAt,
			&item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil
		}
		items = append(items, item)
	}
	return items
}

func (s *MySQLStore) ListCouponsPage(ctx context.Context, accountID string, page int, pageSize int) ([]domain.Coupon, int) {
	items := s.ListCoupons(ctx, accountID)
	total := len(items)
	return paginateSlice(items, page, pageSize), total
}

func (s *MySQLStore) ListUserCoupons(ctx context.Context, accountID string) []domain.UserCoupon {
	rows, err := s.db.QueryContext(ctx, `
		SELECT uc.user_coupon_id, uc.coupon_id, uc.account_id, uc.status, uc.order_id, uc.claimed_at, uc.used_at,
			c.coupon_id, c.name, c.scope, c.merchant_id, c.type, c.threshold_amount, c.discount_amount,
			c.total_count, c.claimed_count, c.per_user_limit, c.start_at, c.end_at, c.status, c.created_at, c.updated_at
		FROM user_coupons uc
		JOIN coupons c ON c.coupon_id = uc.coupon_id
		WHERE uc.account_id = ?
		ORDER BY uc.claimed_at DESC, uc.user_coupon_id DESC
	`, accountID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := make([]domain.UserCoupon, 0)
	for rows.Next() {
		var item domain.UserCoupon
		var usedAt sql.NullTime
		if err := rows.Scan(&item.UserCouponID, &item.CouponID, &item.AccountID, &item.Status, &item.OrderID, &item.ClaimedAt, &usedAt,
			&item.Coupon.CouponID, &item.Coupon.Name, &item.Coupon.Scope, &item.Coupon.MerchantID, &item.Coupon.Type,
			&item.Coupon.ThresholdAmount, &item.Coupon.DiscountAmount, &item.Coupon.TotalCount, &item.Coupon.ClaimedCount,
			&item.Coupon.PerUserLimit, &item.Coupon.StartAt, &item.Coupon.EndAt, &item.Coupon.Status,
			&item.Coupon.CreatedAt, &item.Coupon.UpdatedAt); err != nil {
			return nil
		}
		if usedAt.Valid {
			item.UsedAt = usedAt.Time
		}
		items = append(items, item)
	}
	return items
}

func (s *MySQLStore) ListUserCouponsPage(ctx context.Context, accountID string, page int, pageSize int) ([]domain.UserCoupon, int) {
	items := s.ListUserCoupons(ctx, accountID)
	total := len(items)
	return paginateSlice(items, page, pageSize), total
}

func (s *MySQLStore) ClaimCoupon(ctx context.Context, accountID string, couponID string) (domain.UserCoupon, bool) {
	now := time.Now()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.UserCoupon{}, false
	}
	defer tx.Rollback()
	var coupon domain.Coupon
	// 锁定优惠券行后再判断库存和每人限领，避免高并发下超发。
	err = tx.QueryRowContext(ctx, `
		SELECT coupon_id, name, scope, merchant_id, type, threshold_amount, discount_amount, total_count,
			claimed_count, per_user_limit, start_at, end_at, status, created_at, updated_at
		FROM coupons
		WHERE coupon_id = ? AND status = 'active' AND start_at <= ? AND end_at >= ?
		FOR UPDATE
	`, couponID, now, now).Scan(&coupon.CouponID, &coupon.Name, &coupon.Scope, &coupon.MerchantID, &coupon.Type,
		&coupon.ThresholdAmount, &coupon.DiscountAmount, &coupon.TotalCount, &coupon.ClaimedCount, &coupon.PerUserLimit,
		&coupon.StartAt, &coupon.EndAt, &coupon.Status, &coupon.CreatedAt, &coupon.UpdatedAt)
	if err != nil {
		return domain.UserCoupon{}, false
	}
	if coupon.TotalCount > 0 && coupon.ClaimedCount >= coupon.TotalCount {
		return domain.UserCoupon{}, false
	}
	var owned int
	// 同一个用户对同一张券最多领取 per_user_limit 次。
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM user_coupons WHERE account_id = ? AND coupon_id = ?`, accountID, couponID).Scan(&owned); err != nil {
		return domain.UserCoupon{}, false
	}
	if owned >= coupon.PerUserLimit {
		return domain.UserCoupon{}, false
	}
	item := domain.UserCoupon{UserCouponID: nextID("uc"), CouponID: couponID, AccountID: accountID, Status: "unused", ClaimedAt: now, Coupon: coupon}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO user_coupons (user_coupon_id, coupon_id, account_id, status, claimed_at)
		VALUES (?, ?, ?, ?, ?)
	`, item.UserCouponID, item.CouponID, item.AccountID, item.Status, item.ClaimedAt); err != nil {
		return domain.UserCoupon{}, false
	}
	if _, err := tx.ExecContext(ctx, `UPDATE coupons SET claimed_count = claimed_count + 1, updated_at = ? WHERE coupon_id = ?`, now, couponID); err != nil {
		return domain.UserCoupon{}, false
	}
	if err := tx.Commit(); err != nil {
		return domain.UserCoupon{}, false
	}
	coupon.ClaimedCount++
	item.Coupon = coupon
	return item, true
}
