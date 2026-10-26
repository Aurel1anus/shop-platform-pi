package mysqlstore

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/LYP-leo/xzxg-shop/commerce/internal/domain"
)

func normalizePage(page int, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func pageOffset(page int, pageSize int) int {
	return (page - 1) * pageSize
}

func emptyFallback(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func (s *MySQLStore) countRows(ctx context.Context, table string) int {
	total := 0
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&total); err != nil {
		return 0
	}
	return total
}

func (s *MySQLStore) countRowsWhere(ctx context.Context, table string, where string, args ...any) int {
	total := 0
	query := "SELECT COUNT(*) FROM " + table
	if strings.TrimSpace(where) != "" {
		query += " WHERE " + where
	}
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&total); err != nil {
		return 0
	}
	return total
}

func (s *MySQLStore) queryProductCards(ctx context.Context, query string, args ...any) []domain.ProductCard {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	items := make([]domain.ProductCard, 0)
	for rows.Next() {
		item, err := scanProductCard(rows)
		if err != nil {
			return nil
		}
		items = append(items, item)
	}
	return items
}

func (s *MySQLStore) firstSKU(ctx context.Context, productID string) string {
	var skuID string
	_ = s.db.QueryRowContext(ctx, `
		SELECT sku_id
		FROM product_skus
		WHERE product_id = ?
		ORDER BY sku_id
		LIMIT 1
	`, productID).Scan(&skuID)
	return skuID
}

type productCardScanner interface {
	Scan(dest ...any) error
}

func parseAmount(input string) (float64, error) {
	var value float64
	_, err := fmt.Sscanf(input, "%f", &value)
	return value, err
}

func parseFloat(input string) float64 {
	value, _ := parseAmount(input)
	return value
}

func nextOrderNo(now time.Time) string {
	return "NO" + now.Format("20060102150405") + strings.TrimPrefix(nextID(""), "_")
}

func truncateRunes(input string, limit int) string {
	runes := []rune(strings.TrimSpace(input))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit])
}

func parseAPITime(input string, fallback time.Time) time.Time {
	input = strings.TrimSpace(input)
	if input == "" {
		return fallback
	}
	if value, err := time.Parse(time.RFC3339, input); err == nil {
		return value
	}
	if value, err := time.ParseInLocation("2006-01-02 15:04:05", input, time.Local); err == nil {
		return value
	}
	if value, err := time.ParseInLocation("2006-01-02", input, time.Local); err == nil {
		return value
	}
	return fallback
}

func decodeJSON(input string, output any) {
	if input == "" {
		return
	}
	_ = json.Unmarshal([]byte(input), output)
}

func mustJSON(input any) string {
	payload, err := json.Marshal(input)
	if err != nil {
		return "null"
	}
	return string(payload)
}

func buildCategoryTree(categories []domain.Category) []domain.Category {
	byID := make(map[string]*domain.Category, len(categories))
	rootIDs := make([]string, 0)
	for i := range categories {
		byID[categories[i].CategoryID] = &categories[i]
	}
	for i := range categories {
		item := &categories[i]
		if item.ParentID == "" {
			rootIDs = append(rootIDs, item.CategoryID)
			continue
		}
		parent, ok := byID[item.ParentID]
		if !ok {
			rootIDs = append(rootIDs, item.CategoryID)
			continue
		}
		parent.Children = append(parent.Children, *item)
	}
	roots := make([]domain.Category, 0, len(rootIDs))
	for _, id := range rootIDs {
		roots = append(roots, *byID[id])
	}
	return roots
}

func nextID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

func newAuthToken() (string, error) {
	var bytes [32]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", fmt.Errorf("generate auth token: %w", err)
	}
	return "tok_" + base64.RawURLEncoding.EncodeToString(bytes[:]), nil
}

func authTokenHash(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}

func contentHash(content string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(content)))
	return hex.EncodeToString(sum[:])
}

func splitSQLStatements(input string) []string {
	lines := make([]string, 0)
	for _, line := range strings.Split(input, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") || trimmed == "" {
			continue
		}
		lines = append(lines, line)
	}
	return strings.Split(strings.Join(lines, "\n"), ";")
}

func paginateSlice[T any](items []T, page int, pageSize int) []T {
	page, pageSize = normalizePage(page, pageSize)
	start := pageOffset(page, pageSize)
	if start >= len(items) {
		return []T{}
	}
	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}
