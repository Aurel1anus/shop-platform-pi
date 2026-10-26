package mysqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/LYP-leo/xzxg-shop/commerce/internal/domain"
)

func (s *MySQLStore) ListCategories(ctx context.Context) []domain.Category {
	rows, err := s.db.QueryContext(ctx, `
		SELECT category_id, COALESCE(parent_id, ''), name
		FROM categories
		ORDER BY sort_order, category_id
	`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	categories := make([]domain.Category, 0)
	for rows.Next() {
		var item domain.Category
		if err := rows.Scan(&item.CategoryID, &item.ParentID, &item.Name); err != nil {
			return nil
		}
		item.Children = []domain.Category{}
		categories = append(categories, item)
	}
	return buildCategoryTree(categories)
}

func (s *MySQLStore) ListMerchants(ctx context.Context) []domain.Merchant {
	items, _ := s.queryMerchants(ctx, `
		SELECT merchant_id, name, logo_url, description, service_phone, status
		FROM merchants
		WHERE status = 'active'
		ORDER BY merchant_id
	`)
	return items
}

func (s *MySQLStore) ListMerchantsPage(ctx context.Context, page int, pageSize int) ([]domain.Merchant, int) {
	page, pageSize = normalizePage(page, pageSize)
	total := s.countRowsWhere(ctx, "merchants", "status = 'active'")
	items, _ := s.queryMerchants(ctx, `
		SELECT merchant_id, name, logo_url, description, service_phone, status
		FROM merchants
		WHERE status = 'active'
		ORDER BY merchant_id
		LIMIT ? OFFSET ?
	`, pageSize, pageOffset(page, pageSize))
	return items, total
}

func (s *MySQLStore) queryMerchants(ctx context.Context, query string, args ...any) ([]domain.Merchant, int) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0
	}
	defer rows.Close()

	items := make([]domain.Merchant, 0)
	for rows.Next() {
		var item domain.Merchant
		if err := rows.Scan(&item.MerchantID, &item.Name, &item.LogoURL, &item.Description, &item.ServicePhone, &item.Status); err != nil {
			return nil, 0
		}
		items = append(items, item)
	}
	return items, len(items)
}

func (s *MySQLStore) ListAllMerchantsPage(ctx context.Context, page int, pageSize int) ([]domain.Merchant, int) {
	page, pageSize = normalizePage(page, pageSize)
	total := s.countRows(ctx, "merchants")
	items, _ := s.queryMerchants(ctx, `
		SELECT merchant_id, name, logo_url, description, service_phone, status
		FROM merchants
		ORDER BY merchant_id
		LIMIT ? OFFSET ?
	`, pageSize, pageOffset(page, pageSize))
	return items, total
}

func (s *MySQLStore) UpdateMerchantStatus(ctx context.Context, merchantID string, status string) (domain.Merchant, bool) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE merchants
		SET status = ?, updated_at = ?
		WHERE merchant_id = ?
	`, status, time.Now(), merchantID)
	if err != nil {
		return domain.Merchant{}, false
	}
	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		return domain.Merchant{}, false
	}
	var merchant domain.Merchant
	err = s.db.QueryRowContext(ctx, `
		SELECT merchant_id, name, logo_url, description, service_phone, status
		FROM merchants
		WHERE merchant_id = ?
	`, merchantID).Scan(&merchant.MerchantID, &merchant.Name, &merchant.LogoURL, &merchant.Description, &merchant.ServicePhone, &merchant.Status)
	if err != nil {
		return domain.Merchant{}, false
	}
	return merchant, true
}

func (s *MySQLStore) ListProducts(ctx context.Context, keyword string, categoryID string) []domain.ProductCard {
	query, args := productListQuery(keyword, categoryID)
	query += " ORDER BY p.sort_order, p.product_id"
	return s.queryProductCards(ctx, query, args...)
}

func (s *MySQLStore) ListProductsPage(ctx context.Context, keyword string, categoryID string, page int, pageSize int) ([]domain.ProductCard, int) {
	page, pageSize = normalizePage(page, pageSize)
	query, args := productListQuery(keyword, categoryID)
	countQuery := strings.Replace(query, productCardSelect(), "SELECT COUNT(*)", 1)
	total := 0
	_ = s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	query += " ORDER BY p.sort_order, p.product_id LIMIT ? OFFSET ?"
	args = append(args, pageSize, pageOffset(page, pageSize))
	return s.queryProductCards(ctx, query, args...), total
}

func productListQuery(keyword string, categoryID string) (string, []any) {
	args := make([]any, 0, 3)
	query := productCardSelect() + ` WHERE ` + activeProductCardPredicate()
	if categoryID != "" {
		query += " AND (p.category_id = ? OR c.parent_id = ?)"
		args = append(args, categoryID, categoryID)
	}
	if keyword != "" {
		terms := append([]string{keyword}, strings.Fields(keyword)...)
		clauses := make([]string, 0, len(terms)*5)
		for _, term := range uniqueTerms(terms, 10) {
			like := "%" + term + "%"
			clauses = append(clauses, `p.name LIKE ?`, `p.brand LIKE ?`, `c.name LIKE ?`, `p.tags_json LIKE ?`, `p.selling_points_json LIKE ?`)
			args = append(args, like, like, like, like, like)
		}
		if len(clauses) > 0 {
			query += ` AND (` + strings.Join(clauses, ` OR `) + `)`
		}
	}
	return query, args
}

func (s *MySQLStore) ListAllProducts(ctx context.Context) []domain.ProductCard {
	return s.queryProductCards(ctx, productCardSelect()+" ORDER BY p.sort_order, p.product_id")
}

func (s *MySQLStore) ListAllProductsPage(ctx context.Context, page int, pageSize int) ([]domain.ProductCard, int) {
	page, pageSize = normalizePage(page, pageSize)
	total := 0
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM products p
		JOIN merchants m ON m.merchant_id = p.merchant_id
		LEFT JOIN categories c ON c.category_id = p.category_id
	`).Scan(&total); err != nil {
		return nil, 0
	}
	items := s.queryProductCards(ctx, productCardSelect()+" ORDER BY p.sort_order, p.product_id LIMIT ? OFFSET ?", pageSize, pageOffset(page, pageSize))
	return items, total
}

func (s *MySQLStore) GetProduct(ctx context.Context, productID string) (domain.ProductDetail, bool) {
	row := s.db.QueryRowContext(ctx, productDetailSelect()+` WHERE p.product_id = ?`, productID)
	product, err := scanProductDetail(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ProductDetail{}, false
	}
	if err != nil {
		return domain.ProductDetail{}, false
	}
	return product, true
}

func (s *MySQLStore) CreateProduct(ctx context.Context, input domain.ProductUpsertInput) (domain.ProductDetail, error) {
	productID := input.ProductID
	if productID == "" {
		productID = nextID("prod")
	}
	input.ProductID = productID
	normalized := input
	if err := s.insertProduct(ctx, normalized); err != nil {
		return domain.ProductDetail{}, err
	}
	if err := s.upsertDefaultSKU(ctx, normalized); err != nil {
		return domain.ProductDetail{}, err
	}
	product, ok := s.GetProduct(ctx, productID)
	if !ok {
		return domain.ProductDetail{}, errors.New("created product not found")
	}
	return product, nil
}

func (s *MySQLStore) insertProduct(ctx context.Context, input domain.ProductUpsertInput) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO products (
			product_id, merchant_id, name, brand, category_id, image_url, image_urls_json,
			price, market_price, stock_quantity, stock_status, tags_json, selling_points_json,
			recommend_reason, risk_notes_json, attributes_json, suitable_for_json,
			not_suitable_for_json, description, status, sort_order, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'active', 100, ?, ?)
	`,
		input.ProductID,
		input.MerchantID,
		input.Name,
		input.Brand,
		input.CategoryID,
		input.ImageURL,
		mustJSON([]string{input.ImageURL}),
		input.Price,
		input.MarketPrice,
		input.StockQuantity,
		input.StockStatus,
		mustJSON(input.Tags),
		mustJSON(input.SellingPoints),
		input.RecommendReason,
		mustJSON(input.RiskNotes),
		mustJSON([]domain.ProductAttribute{}),
		mustJSON([]string{}),
		mustJSON([]string{}),
		input.Description,
		time.Now(),
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("insert product: %w", err)
	}
	return nil
}

func (s *MySQLStore) upsertDefaultSKU(ctx context.Context, input domain.ProductUpsertInput) error {
	skuID := "sku_" + input.ProductID
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO product_skus (sku_id, product_id, sku_name, price, stock_quantity, stock_status, specs_json, is_default, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, TRUE, ?, ?)
		ON DUPLICATE KEY UPDATE sku_name = VALUES(sku_name), price = VALUES(price),
			stock_quantity = VALUES(stock_quantity), stock_status = VALUES(stock_status),
			specs_json = VALUES(specs_json), is_default = TRUE, updated_at = VALUES(updated_at)
	`, skuID, input.ProductID, input.Name+" 默认款", input.Price, input.StockQuantity, input.StockStatus, mustJSON(map[string]string{"版本": "默认款"}), time.Now(), time.Now())
	if err != nil {
		return fmt.Errorf("upsert default sku: %w", err)
	}
	return nil
}

func (s *MySQLStore) UpdateProduct(ctx context.Context, productID string, input domain.ProductUpsertInput) (domain.ProductDetail, bool) {
	input.ProductID = productID
	normalized := input
	result, err := s.db.ExecContext(ctx, `
		UPDATE products
		SET name = ?, brand = ?, category_id = ?, image_url = ?, image_urls_json = ?,
			price = ?, market_price = ?, stock_quantity = ?, stock_status = ?,
			tags_json = ?, selling_points_json = ?, recommend_reason = ?,
			risk_notes_json = ?, attributes_json = ?, suitable_for_json = ?,
			not_suitable_for_json = ?, description = ?, updated_at = ?
		WHERE product_id = ? AND merchant_id = ?
	`,
		normalized.Name,
		normalized.Brand,
		normalized.CategoryID,
		normalized.ImageURL,
		mustJSON([]string{normalized.ImageURL}),
		normalized.Price,
		normalized.MarketPrice,
		normalized.StockQuantity,
		normalized.StockStatus,
		mustJSON(normalized.Tags),
		mustJSON(normalized.SellingPoints),
		normalized.RecommendReason,
		mustJSON(normalized.RiskNotes),
		mustJSON([]domain.ProductAttribute{}),
		mustJSON([]string{}),
		mustJSON([]string{}),
		normalized.Description,
		time.Now(),
		productID,
		normalized.MerchantID,
	)
	if err != nil {
		return domain.ProductDetail{}, false
	}
	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		return domain.ProductDetail{}, false
	}
	if err := s.upsertDefaultSKU(ctx, normalized); err != nil {
		return domain.ProductDetail{}, false
	}
	product, ok := s.GetProduct(ctx, productID)
	return product, ok
}

func (s *MySQLStore) UpdateProductStatus(ctx context.Context, merchantID string, productID string, status string) (domain.ProductDetail, bool) {
	args := []any{status, time.Now(), productID}
	query := `UPDATE products SET status = ?, updated_at = ? WHERE product_id = ?`
	if merchantID != "" {
		query += ` AND merchant_id = ?`
		args = append(args, merchantID)
	}
	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return domain.ProductDetail{}, false
	}
	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		return domain.ProductDetail{}, false
	}
	product, ok := s.GetProduct(ctx, productID)
	return product, ok
}

func (s *MySQLStore) ListProductSKUs(ctx context.Context, productID string) []domain.ProductSKU {
	rows, err := s.db.QueryContext(ctx, `
		SELECT sku_id, product_id, sku_name, price, stock_quantity, stock_status, specs_json
		FROM product_skus
		WHERE product_id = ?
		ORDER BY sku_id
	`, productID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	items := make([]domain.ProductSKU, 0)
	for rows.Next() {
		item, err := scanSKU(rows)
		if err != nil {
			return nil
		}
		items = append(items, item)
	}
	return items
}

func scanProductCard(scanner productCardScanner) (domain.ProductCard, error) {
	var item domain.ProductCard
	var tagsJSON string
	var sellingPointsJSON string
	var riskNotesJSON string
	err := scanner.Scan(
		&item.ProductID,
		&item.SkuID,
		&item.MerchantID,
		&item.MerchantName,
		&item.Name,
		&item.Brand,
		&item.CategoryID,
		&item.ImageURL,
		&item.Price,
		&item.MarketPrice,
		&item.StockStatus,
		&item.Status,
		&tagsJSON,
		&sellingPointsJSON,
		&item.RecommendReason,
		&riskNotesJSON,
	)
	if err != nil {
		return domain.ProductCard{}, err
	}
	decodeJSON(tagsJSON, &item.Tags)
	decodeJSON(sellingPointsJSON, &item.SellingPoints)
	decodeJSON(riskNotesJSON, &item.RiskNotes)
	return item, nil
}

func scanProductDetail(scanner productCardScanner) (domain.ProductDetail, error) {
	var product domain.ProductDetail
	var tagsJSON string
	var sellingPointsJSON string
	var riskNotesJSON string
	var imageURLsJSON string
	var attributesJSON string
	var suitableForJSON string
	var notSuitableForJSON string
	err := scanner.Scan(
		&product.ProductID,
		&product.SkuID,
		&product.MerchantID,
		&product.MerchantName,
		&product.Name,
		&product.Brand,
		&product.CategoryID,
		&product.ImageURL,
		&product.Price,
		&product.MarketPrice,
		&product.StockStatus,
		&product.Status,
		&tagsJSON,
		&sellingPointsJSON,
		&product.RecommendReason,
		&riskNotesJSON,
		&imageURLsJSON,
		&product.StockQuantity,
		&attributesJSON,
		&suitableForJSON,
		&notSuitableForJSON,
		&product.Description,
	)
	if err != nil {
		return domain.ProductDetail{}, err
	}
	decodeJSON(tagsJSON, &product.Tags)
	decodeJSON(sellingPointsJSON, &product.SellingPoints)
	decodeJSON(riskNotesJSON, &product.RiskNotes)
	decodeJSON(imageURLsJSON, &product.ImageURLs)
	decodeJSON(attributesJSON, &product.Attributes)
	decodeJSON(suitableForJSON, &product.SuitableFor)
	decodeJSON(notSuitableForJSON, &product.NotSuitableFor)
	return product, nil
}

func scanSKU(scanner productCardScanner) (domain.ProductSKU, error) {
	var item domain.ProductSKU
	var specsJSON string
	err := scanner.Scan(&item.SkuID, &item.ProductID, &item.SkuName, &item.Price, &item.StockQuantity, &item.StockStatus, &specsJSON)
	if err != nil {
		return domain.ProductSKU{}, err
	}
	decodeJSON(specsJSON, &item.Specs)
	return item, nil
}

func productCardSelect() string {
	return `
		SELECT p.product_id, COALESCE(ps.sku_id, ''), p.merchant_id, m.name,
			p.name, p.brand, p.category_id, p.image_url, p.price, p.market_price,
			p.stock_status, p.status, p.tags_json, p.selling_points_json, p.recommend_reason,
			p.risk_notes_json
		FROM products p
		JOIN merchants m ON m.merchant_id = p.merchant_id
		LEFT JOIN categories c ON c.category_id = p.category_id
		LEFT JOIN product_skus ps ON ps.product_id = p.product_id AND ps.is_default = TRUE
	`
}

func productDetailSelect() string {
	return `
		SELECT p.product_id, COALESCE(ps.sku_id, ''), p.merchant_id, m.name,
			p.name, p.brand, p.category_id, p.image_url, p.price, p.market_price,
			p.stock_status, p.status, p.tags_json, p.selling_points_json, p.recommend_reason,
			p.risk_notes_json, p.image_urls_json, p.stock_quantity,
			p.attributes_json, p.suitable_for_json, p.not_suitable_for_json, p.description
		FROM products p
		JOIN merchants m ON m.merchant_id = p.merchant_id
		LEFT JOIN categories c ON c.category_id = p.category_id
		LEFT JOIN product_skus ps ON ps.product_id = p.product_id AND ps.is_default = TRUE
	`
}

func activeProductCardPredicate() string {
	return "p.status = '" + domain.ProductStatusActive + "' AND m.status = '" + domain.MerchantStatusActive + "'"
}

func uniqueTerms(input []string, limit int) []string {
	seen := make(map[string]bool)
	terms := make([]string, 0, limit)
	for _, term := range input {
		term = strings.TrimSpace(strings.ToLower(term))
		if term == "" || seen[term] {
			continue
		}
		seen[term] = true
		terms = append(terms, term)
		if len(terms) == limit {
			break
		}
	}
	return terms
}
