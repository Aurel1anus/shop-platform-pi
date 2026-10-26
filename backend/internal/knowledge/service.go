package knowledge

import (
	"context"
	"errors"
	"strings"

	"github.com/LYP-leo/xzxg-shop/commerce/internal/domain"
)

type Repository interface {
	CreateMerchantDocument(ctx context.Context, input domain.KnowledgeDocumentInput) (domain.KnowledgeDocument, error)
	ListAllDocumentsPage(ctx context.Context, page int, pageSize int) ([]domain.KnowledgeDocument, int)
	ListMerchantDocumentsPage(ctx context.Context, merchantID string, page int, pageSize int) ([]domain.KnowledgeDocument, int)
}

type Service struct{ repo Repository }

func New(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) CreateMerchantDocument(ctx context.Context, input domain.KnowledgeDocumentInput) (domain.KnowledgeDocument, error) {
	input.MerchantID = strings.TrimSpace(input.MerchantID)
	input.Title = strings.TrimSpace(input.Title)
	input.Content = strings.TrimSpace(input.Content)
	if input.MerchantID == "" || input.Title == "" || input.Content == "" {
		return domain.KnowledgeDocument{}, errors.New("商家、资料标题和内容不能为空")
	}
	if input.DocType == "" {
		input.DocType = "product_detail"
	}
	return s.repo.CreateMerchantDocument(ctx, input)
}

func (s *Service) ListAllDocumentsPage(ctx context.Context, page int, pageSize int) ([]domain.KnowledgeDocument, int) {
	return s.repo.ListAllDocumentsPage(ctx, page, pageSize)
}

func (s *Service) ListMerchantDocumentsPage(ctx context.Context, merchantID string, page int, pageSize int) ([]domain.KnowledgeDocument, int) {
	return s.repo.ListMerchantDocumentsPage(ctx, merchantID, page, pageSize)
}
