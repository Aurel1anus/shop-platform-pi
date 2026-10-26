package mysqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/LYP-leo/xzxg-shop/commerce/internal/domain"
)

func (s *MySQLStore) ListMerchantDocuments(ctx context.Context, merchantID string) []domain.KnowledgeDocument {
	items, _ := s.ListMerchantDocumentsPage(ctx, merchantID, 1, 100)
	return items
}

func (s *MySQLStore) ListMerchantDocumentsPage(ctx context.Context, merchantID string, page int, pageSize int) ([]domain.KnowledgeDocument, int) {
	page, pageSize = normalizePage(page, pageSize)
	total := s.countRowsWhere(ctx, "knowledge_documents", "merchant_id = ?", merchantID)
	rows, err := s.db.QueryContext(ctx, `
		SELECT document_id, merchant_id, title, doc_type, status, chunk_count, source_url, content_hash, content, metadata_json, created_at
		FROM knowledge_documents
		WHERE merchant_id = ?
		ORDER BY created_at DESC, document_id DESC
		LIMIT ? OFFSET ?
	`, merchantID, pageSize, pageOffset(page, pageSize))
	if err != nil {
		return nil, 0
	}
	defer rows.Close()

	items := make([]domain.KnowledgeDocument, 0)
	for rows.Next() {
		var item domain.KnowledgeDocument
		var metadataJSON []byte
		if err := rows.Scan(&item.DocumentID, &item.MerchantID, &item.Title, &item.DocType, &item.Status, &item.ChunkCount, &item.SourceURL, &item.ContentHash, &item.Content, &metadataJSON, &item.CreatedAt); err != nil {
			return nil, 0
		}
		_ = json.Unmarshal(metadataJSON, &item.Metadata)
		items = append(items, item)
	}
	return items, total
}

func (s *MySQLStore) ListAllDocuments(ctx context.Context) []domain.KnowledgeDocument {
	rows, err := s.db.QueryContext(ctx, `
		SELECT document_id, merchant_id, title, doc_type, status, chunk_count, source_url, content_hash, content, metadata_json, created_at
		FROM knowledge_documents
		ORDER BY created_at DESC, document_id DESC
	`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	items := make([]domain.KnowledgeDocument, 0)
	for rows.Next() {
		var item domain.KnowledgeDocument
		var metadataJSON []byte
		if err := rows.Scan(&item.DocumentID, &item.MerchantID, &item.Title, &item.DocType, &item.Status, &item.ChunkCount, &item.SourceURL, &item.ContentHash, &item.Content, &metadataJSON, &item.CreatedAt); err != nil {
			return nil
		}
		_ = json.Unmarshal(metadataJSON, &item.Metadata)
		items = append(items, item)
	}
	return items
}

func (s *MySQLStore) ListAllDocumentsPage(ctx context.Context, page int, pageSize int) ([]domain.KnowledgeDocument, int) {
	page, pageSize = normalizePage(page, pageSize)
	total := s.countRows(ctx, "knowledge_documents")
	rows, err := s.db.QueryContext(ctx, `
		SELECT document_id, merchant_id, title, doc_type, status, chunk_count, source_url, content_hash, content, metadata_json, created_at
		FROM knowledge_documents
		ORDER BY created_at DESC, document_id DESC
		LIMIT ? OFFSET ?
	`, pageSize, pageOffset(page, pageSize))
	if err != nil {
		return nil, 0
	}
	defer rows.Close()

	items := make([]domain.KnowledgeDocument, 0)
	for rows.Next() {
		var item domain.KnowledgeDocument
		var metadataJSON []byte
		if err := rows.Scan(&item.DocumentID, &item.MerchantID, &item.Title, &item.DocType, &item.Status, &item.ChunkCount, &item.SourceURL, &item.ContentHash, &item.Content, &metadataJSON, &item.CreatedAt); err != nil {
			return nil, 0
		}
		_ = json.Unmarshal(metadataJSON, &item.Metadata)
		items = append(items, item)
	}
	return items, total
}

func (s *MySQLStore) CreateMerchantDocument(ctx context.Context, input domain.KnowledgeDocumentInput) (domain.KnowledgeDocument, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Content = strings.TrimSpace(input.Content)
	input.DocType = strings.TrimSpace(input.DocType)
	input.SourceURL = strings.TrimSpace(input.SourceURL)
	input.ContentHash = strings.TrimSpace(input.ContentHash)
	if input.ContentHash == "" {
		input.ContentHash = contentHash(input.Content)
	}
	if input.ContentHash != "" && !input.ForceReindex {
		if existing, ok := s.getKnowledgeDocumentByHash(ctx, input.MerchantID, input.ContentHash); ok {
			return existing, nil
		}
	}
	metadataJSON := "{}"
	if len(input.Metadata) > 0 {
		if data, err := json.Marshal(input.Metadata); err == nil {
			metadataJSON = string(data)
		}
	}
	now := time.Now()
	document := domain.KnowledgeDocument{
		Content: input.Content, Metadata: input.Metadata,
		DocumentID:  nextID("doc"),
		MerchantID:  input.MerchantID,
		Title:       input.Title,
		DocType:     input.DocType,
		Status:      "pending_index",
		ChunkCount:  0,
		SourceURL:   input.SourceURL,
		ContentHash: input.ContentHash,
		CreatedAt:   now,
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.KnowledgeDocument{}, err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO knowledge_documents (
			document_id, merchant_id, title, doc_type, content, status, chunk_count, source_url, content_hash, metadata_json, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, document.DocumentID, document.MerchantID, document.Title, document.DocType, input.Content, document.Status, document.ChunkCount, document.SourceURL, document.ContentHash, metadataJSON, now, now)
	if err != nil {
		return domain.KnowledgeDocument{}, fmt.Errorf("insert knowledge document: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return domain.KnowledgeDocument{}, err
	}
	return document, nil
}

func (s *MySQLStore) getKnowledgeDocumentByHash(ctx context.Context, merchantID string, hash string) (domain.KnowledgeDocument, bool) {
	var item domain.KnowledgeDocument
	var metadataJSON []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT document_id, merchant_id, title, doc_type, status, chunk_count, source_url, content_hash, content, metadata_json, created_at
		FROM knowledge_documents
		WHERE merchant_id = ? AND content_hash = ? AND content_hash <> ''
		ORDER BY created_at DESC, document_id DESC
		LIMIT 1
	`, merchantID, hash).Scan(&item.DocumentID, &item.MerchantID, &item.Title, &item.DocType, &item.Status, &item.ChunkCount, &item.SourceURL, &item.ContentHash, &item.Content, &metadataJSON, &item.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.KnowledgeDocument{}, false
	}
	_ = json.Unmarshal(metadataJSON, &item.Metadata)
	return item, err == nil
}
