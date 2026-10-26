package files

import (
	"context"

	"github.com/LYP-leo/xzxg-shop/commerce/internal/domain"
)

type Repository interface {
	CreateStoredFile(ctx context.Context, input domain.StoredFileInput) (domain.StoredFile, error)
	GetStoredFile(ctx context.Context, fileID string) (domain.StoredFile, bool)
}

type Service struct{ repo Repository }

func New(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) CreateStoredFile(ctx context.Context, input domain.StoredFileInput) (domain.StoredFile, error) {
	return s.repo.CreateStoredFile(ctx, input)
}

func (s *Service) GetStoredFile(ctx context.Context, fileID string) (domain.StoredFile, bool) {
	return s.repo.GetStoredFile(ctx, fileID)
}
