CREATE TABLE IF NOT EXISTS merchants (
  merchant_id VARCHAR(64) PRIMARY KEY,
  name VARCHAR(128) NOT NULL,
  logo_url VARCHAR(512) NOT NULL DEFAULT '',
  description TEXT NOT NULL,
  service_phone VARCHAR(64) NOT NULL DEFAULT '',
  status VARCHAR(32) NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS categories (
  category_id VARCHAR(64) PRIMARY KEY,
  parent_id VARCHAR(64) NOT NULL DEFAULT '',
  name VARCHAR(128) NOT NULL,
  sort_order INT NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_categories_parent_id (parent_id)
);

CREATE TABLE IF NOT EXISTS products (
  product_id VARCHAR(64) PRIMARY KEY,
  merchant_id VARCHAR(64) NOT NULL,
  name VARCHAR(128) NOT NULL,
  brand VARCHAR(64) NOT NULL,
  category_id VARCHAR(64) NOT NULL,
  image_url VARCHAR(512) NOT NULL,
  image_urls_json JSON NOT NULL,
  price DECIMAL(10, 2) NOT NULL,
  market_price DECIMAL(10, 2) NOT NULL,
  stock_quantity INT NOT NULL,
  stock_status VARCHAR(32) NOT NULL,
  tags_json JSON NOT NULL,
  selling_points_json JSON NOT NULL,
  recommend_reason TEXT NOT NULL,
  risk_notes_json JSON NOT NULL,
  attributes_json JSON NOT NULL,
  suitable_for_json JSON NOT NULL,
  not_suitable_for_json JSON NOT NULL,
  description TEXT NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  sort_order INT NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_products_category_id (category_id),
  INDEX idx_products_merchant_id (merchant_id),
  INDEX idx_products_status (status)
);

CREATE TABLE IF NOT EXISTS product_skus (
  sku_id VARCHAR(64) PRIMARY KEY,
  product_id VARCHAR(64) NOT NULL,
  sku_name VARCHAR(128) NOT NULL,
  price DECIMAL(10, 2) NOT NULL,
  stock_quantity INT NOT NULL,
  stock_status VARCHAR(32) NOT NULL,
  specs_json JSON NOT NULL,
  is_default BOOLEAN NOT NULL DEFAULT FALSE,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_product_skus_product_id (product_id)
);

CREATE TABLE IF NOT EXISTS knowledge_documents (
  document_id VARCHAR(64) PRIMARY KEY,
  merchant_id VARCHAR(64) NOT NULL,
  title VARCHAR(256) NOT NULL,
  doc_type VARCHAR(64) NOT NULL,
  content MEDIUMTEXT NOT NULL,
  status VARCHAR(32) NOT NULL,
  chunk_count INT NOT NULL DEFAULT 0,
  source_url VARCHAR(1024) NOT NULL DEFAULT '',
  content_hash VARCHAR(64) NOT NULL DEFAULT '',
  metadata_json JSON NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_knowledge_documents_merchant_id (merchant_id),
  INDEX idx_knowledge_documents_status (status),
  INDEX idx_knowledge_documents_content_hash (content_hash)
);

CREATE TABLE IF NOT EXISTS accounts (
  account_id VARCHAR(64) PRIMARY KEY,
  username VARCHAR(64) NOT NULL,
  password_hash VARCHAR(128) NOT NULL,
  display_name VARCHAR(128) NOT NULL,
  avatar_url VARCHAR(512) NOT NULL DEFAULT '',
  phone VARCHAR(32) NOT NULL DEFAULT '',
  email VARCHAR(128) NOT NULL DEFAULT '',
  role VARCHAR(32) NOT NULL,
  merchant_id VARCHAR(64) NOT NULL DEFAULT '',
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  deleted_at DATETIME NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_accounts_username (username),
  INDEX idx_accounts_status (status),
  INDEX idx_accounts_role (role)
);

CREATE TABLE IF NOT EXISTS auth_tokens (
  token VARCHAR(96) PRIMARY KEY,
  account_id VARCHAR(64) NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  expires_at DATETIME NOT NULL,
  INDEX idx_auth_tokens_account_id (account_id),
  INDEX idx_auth_tokens_expires_at (expires_at)
);

CREATE TABLE IF NOT EXISTS cart_items (
  cart_item_id VARCHAR(64) PRIMARY KEY,
  account_id VARCHAR(64) NOT NULL DEFAULT '',
  product_id VARCHAR(64) NOT NULL,
  sku_id VARCHAR(64) NOT NULL DEFAULT '',
  quantity INT NOT NULL,
  selected BOOLEAN NOT NULL DEFAULT TRUE,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_cart_account_product_sku (account_id, product_id, sku_id),
  INDEX idx_cart_items_account_id (account_id),
  INDEX idx_cart_items_product_id (product_id)
);

CREATE TABLE IF NOT EXISTS orders (
  order_id VARCHAR(64) PRIMARY KEY,
  order_no VARCHAR(64) NOT NULL DEFAULT '',
  account_id VARCHAR(64) NOT NULL,
  merchant_id VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL,
  total_amount DECIMAL(10, 2) NOT NULL,
  discount_amount DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
  pay_amount DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
  payment_deadline_at DATETIME NULL,
  paid_at DATETIME NULL,
  closed_at DATETIME NULL,
  completed_at DATETIME NULL,
  cancel_reason VARCHAR(256) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_orders_order_no (order_no),
  INDEX idx_orders_account_id (account_id),
  INDEX idx_orders_merchant_id (merchant_id),
  INDEX idx_orders_status (status),
  INDEX idx_orders_payment_deadline_at (payment_deadline_at)
);

CREATE TABLE IF NOT EXISTS payments (
  payment_id VARCHAR(64) PRIMARY KEY,
  order_id VARCHAR(64) NOT NULL,
  account_id VARCHAR(64) NOT NULL,
  amount DECIMAL(10, 2) NOT NULL,
  status VARCHAR(32) NOT NULL,
  method VARCHAR(32) NOT NULL,
  transaction_no VARCHAR(96) NOT NULL DEFAULT '',
  expires_at DATETIME NOT NULL,
  paid_at DATETIME NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_payments_order_id (order_id),
  INDEX idx_payments_account_id (account_id),
  INDEX idx_payments_status (status),
  INDEX idx_payments_expires_at (expires_at)
);

CREATE TABLE IF NOT EXISTS promotion_rules (
  promotion_id VARCHAR(64) PRIMARY KEY,
  name VARCHAR(128) NOT NULL,
  scope VARCHAR(32) NOT NULL,
  merchant_id VARCHAR(64) NOT NULL DEFAULT '',
  product_id VARCHAR(64) NOT NULL DEFAULT '',
  category_id VARCHAR(64) NOT NULL DEFAULT '',
  type VARCHAR(32) NOT NULL,
  threshold_amount DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
  discount_amount DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
  discount_rate DECIMAL(5, 4) NOT NULL DEFAULT 0.0000,
  stackable BOOLEAN NOT NULL DEFAULT TRUE,
  start_at DATETIME NOT NULL,
  end_at DATETIME NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_promotion_rules_scope (scope),
  INDEX idx_promotion_rules_merchant_id (merchant_id),
  INDEX idx_promotion_rules_status (status),
  INDEX idx_promotion_rules_time (start_at, end_at)
);

CREATE TABLE IF NOT EXISTS coupons (
  coupon_id VARCHAR(64) PRIMARY KEY,
  name VARCHAR(128) NOT NULL,
  scope VARCHAR(32) NOT NULL,
  merchant_id VARCHAR(64) NOT NULL DEFAULT '',
  type VARCHAR(32) NOT NULL,
  threshold_amount DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
  discount_amount DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
  total_count INT NOT NULL DEFAULT 0,
  claimed_count INT NOT NULL DEFAULT 0,
  per_user_limit INT NOT NULL DEFAULT 1,
  start_at DATETIME NOT NULL,
  end_at DATETIME NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_coupons_scope (scope),
  INDEX idx_coupons_merchant_id (merchant_id),
  INDEX idx_coupons_status (status),
  INDEX idx_coupons_time (start_at, end_at)
);

CREATE TABLE IF NOT EXISTS user_coupons (
  user_coupon_id VARCHAR(64) PRIMARY KEY,
  coupon_id VARCHAR(64) NOT NULL,
  account_id VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'unused',
  order_id VARCHAR(64) NOT NULL DEFAULT '',
  claimed_at DATETIME NOT NULL,
  used_at DATETIME NULL,
  INDEX idx_user_coupons_account_id (account_id),
  INDEX idx_user_coupons_coupon_id (coupon_id),
  INDEX idx_user_coupons_status (status)
);

CREATE TABLE IF NOT EXISTS product_reviews (
  review_id VARCHAR(64) PRIMARY KEY,
  order_id VARCHAR(64) NOT NULL,
  order_item_id VARCHAR(64) NOT NULL,
  product_id VARCHAR(64) NOT NULL,
  sku_id VARCHAR(64) NOT NULL DEFAULT '',
  account_id VARCHAR(64) NOT NULL,
  rating INT NOT NULL,
  content TEXT NOT NULL,
  tags_json JSON NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'visible',
  merchant_reply TEXT,
  merchant_replied_at DATETIME NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_product_reviews_order_item (order_item_id),
  INDEX idx_product_reviews_product_id (product_id),
  INDEX idx_product_reviews_account_id (account_id),
  INDEX idx_product_reviews_status (status)
);

CREATE TABLE IF NOT EXISTS order_items (
  order_item_id VARCHAR(64) PRIMARY KEY,
  order_id VARCHAR(64) NOT NULL,
  product_id VARCHAR(64) NOT NULL,
  sku_id VARCHAR(64) NOT NULL DEFAULT '',
  name VARCHAR(128) NOT NULL,
  image_url VARCHAR(512) NOT NULL,
  price DECIMAL(10, 2) NOT NULL,
  quantity INT NOT NULL,
  merchant_id VARCHAR(64) NOT NULL,
  merchant_name VARCHAR(128) NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_order_items_order_id (order_id),
  INDEX idx_order_items_product_id (product_id)
);

CREATE TABLE IF NOT EXISTS stored_files (
			file_id VARCHAR(64) PRIMARY KEY,
			account_id VARCHAR(64) NOT NULL DEFAULT '',
			object_key VARCHAR(512) NOT NULL,
			url VARCHAR(512) NOT NULL,
			mime_type VARCHAR(128) NOT NULL,
			size_bytes BIGINT NOT NULL DEFAULT 0,
			content_hash VARCHAR(128) NOT NULL,
			storage_provider VARCHAR(32) NOT NULL DEFAULT 'minio',
			source_url VARCHAR(1024) NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE KEY uk_stored_files_object_key (object_key),
			INDEX idx_stored_files_account_id (account_id),
			INDEX idx_stored_files_content_hash (content_hash)
		);
