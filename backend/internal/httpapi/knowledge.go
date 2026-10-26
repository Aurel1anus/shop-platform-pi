package httpapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/LYP-leo/xzxg-shop/commerce/internal/domain"
	"github.com/LYP-leo/xzxg-shop/commerce/internal/knowledge/ingest"
)

func (s *Server) handleListMerchantDocuments(w http.ResponseWriter, r *http.Request) {
	account, ok := s.requireMerchant(w, r)
	if !ok {
		return
	}
	page, pageSize := readPagination(r)
	items, total := s.knowledge.ListMerchantDocumentsPage(r.Context(), account.MerchantID, page, pageSize)
	writeJSON(w, http.StatusOK, pagedPayload(items, page, pageSize, total))
}

func (s *Server) handleCreateMerchantDocument(w http.ResponseWriter, r *http.Request) {
	account, ok := s.requireMerchant(w, r)
	if !ok {
		return
	}
	var request domain.KnowledgeDocumentInput
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "请求 JSON 不合法")
		return
	}
	if strings.TrimSpace(request.Title) == "" || strings.TrimSpace(request.Content) == "" {
		writeError(w, http.StatusBadRequest, "empty_document", "资料标题和内容不能为空")
		return
	}
	if request.DocType == "" {
		request.DocType = "product_detail"
	}
	request.MerchantID = account.MerchantID
	document, err := s.knowledge.CreateMerchantDocument(r.Context(), request)
	if err != nil {
		s.logger.Error("create merchant document failed", "error", err, "merchant_id", account.MerchantID)
		writeError(w, http.StatusInternalServerError, "create_document_failed", "上传资料失败")
		return
	}
	writeJSON(w, http.StatusOK, document)
}

func (s *Server) handleCreateMerchantUnstructuredIngestion(w http.ResponseWriter, r *http.Request) {
	account, ok := s.requireMerchant(w, r)
	if !ok {
		return
	}
	var request domain.UnstructuredIngestRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "请求 JSON 不合法")
		return
	}
	request.MerchantID = account.MerchantID
	s.createUnstructuredIngestion(w, r, request)
}

func (s *Server) handleCreateAdminUnstructuredIngestion(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var request domain.UnstructuredIngestRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "请求 JSON 不合法")
		return
	}
	if strings.TrimSpace(request.MerchantID) == "" {
		writeError(w, http.StatusBadRequest, "empty_merchant_id", "管理员入库必须指定 merchant_id")
		return
	}
	s.createUnstructuredIngestion(w, r, request)
}

func (s *Server) createUnstructuredIngestion(w http.ResponseWriter, r *http.Request, request domain.UnstructuredIngestRequest) {
	parsed, err := ingest.Parse(r.Context(), request)
	if err != nil {
		writeError(w, http.StatusBadRequest, "parse_ingestion_failed", err.Error())
		return
	}
	hash := sha256.Sum256([]byte(parsed.Content))
	startedAt := time.Now()
	contentHash := hex.EncodeToString(hash[:])
	document, err := s.knowledge.CreateMerchantDocument(r.Context(), domain.KnowledgeDocumentInput{
		MerchantID:   strings.TrimSpace(request.MerchantID),
		Title:        parsed.Title,
		DocType:      parsed.DocType,
		Content:      parsed.Content,
		SourceURL:    parsed.SourceURL,
		ContentHash:  contentHash,
		Metadata:     parsed.Metadata,
		ForceReindex: request.ForceReindex,
	})
	if err != nil {
		s.logger.Error("create unstructured ingestion failed", "error", err, "merchant_id", request.MerchantID)
		writeError(w, http.StatusInternalServerError, "create_ingestion_failed", "非结构化资料入库失败")
		return
	}
	writeJSON(w, http.StatusCreated, domain.UnstructuredIngestResult{
		Document:  document,
		Duplicate: document.ContentHash == contentHash && document.CreatedAt.Before(startedAt),
		TextRunes: parsed.TextRunes,
	})
}

func (s *Server) handleListAdminDocuments(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	page, pageSize := readPagination(r)
	items, total := s.knowledge.ListAllDocumentsPage(r.Context(), page, pageSize)
	writeJSON(w, http.StatusOK, pagedPayload(items, page, pageSize, total))
}
