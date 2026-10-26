package mysqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/LYP-leo/xzxg-shop/commerce/internal/domain"
)

func (s *MySQLStore) CreateStoredFile(ctx context.Context, input domain.StoredFileInput) (domain.StoredFile, error) {
	if input.FileID == "" {
		input.FileID = nextID("file")
	}
	if input.StorageProvider == "" {
		input.StorageProvider = "minio"
	}
	now := time.Now()
	file := domain.StoredFile{
		FileID:          input.FileID,
		AccountID:       input.AccountID,
		ObjectKey:       input.ObjectKey,
		URL:             input.URL,
		MimeType:        input.MimeType,
		SizeBytes:       input.SizeBytes,
		ContentHash:     input.ContentHash,
		StorageProvider: input.StorageProvider,
		SourceURL:       input.SourceURL,
		CreatedAt:       now,
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO stored_files (file_id, account_id, object_key, url, mime_type, size_bytes, content_hash, storage_provider, source_url, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, file.FileID, file.AccountID, file.ObjectKey, file.URL, file.MimeType, file.SizeBytes, file.ContentHash, file.StorageProvider, file.SourceURL, file.CreatedAt)
	if err != nil {
		return domain.StoredFile{}, fmt.Errorf("insert stored file: %w", err)
	}
	return file, nil
}

func (s *MySQLStore) GetStoredFile(ctx context.Context, fileID string) (domain.StoredFile, bool) {
	var file domain.StoredFile
	err := s.db.QueryRowContext(ctx, `
		SELECT file_id, account_id, object_key, url, mime_type, size_bytes, content_hash, storage_provider, source_url, created_at
		FROM stored_files
		WHERE file_id = ?
	`, fileID).Scan(
		&file.FileID,
		&file.AccountID,
		&file.ObjectKey,
		&file.URL,
		&file.MimeType,
		&file.SizeBytes,
		&file.ContentHash,
		&file.StorageProvider,
		&file.SourceURL,
		&file.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.StoredFile{}, false
	}
	return file, err == nil
}
