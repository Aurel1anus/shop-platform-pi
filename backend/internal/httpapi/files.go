package httpapi

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/LYP-leo/xzxg-shop/commerce/internal/domain"
	"github.com/LYP-leo/xzxg-shop/commerce/internal/platform/objectstore"
)

func (s *Server) handleUploadFile(w http.ResponseWriter, r *http.Request) {
	account, ok := accountFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "请先登录")
		return
	}
	maxBytes := int64FromMap(s.configs.GetMap(r.Context()), "files.max_upload_bytes", 10<<20)
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	if err := r.ParseMultipartForm(maxBytes); err != nil {
		writeError(w, http.StatusBadRequest, "bad_multipart", "文件上传请求不合法或文件过大")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "empty_file", "缺少 file 字段")
		return
	}
	defer file.Close()

	buffer, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "read_file_failed", "读取文件失败")
		return
	}
	if int64(len(buffer)) > maxBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "file_too_large", "文件超过上传大小限制")
		return
	}
	mimeType := http.DetectContentType(buffer)
	if !strings.HasPrefix(mimeType, "image/") && !strings.HasPrefix(mimeType, "application/pdf") {
		writeError(w, http.StatusBadRequest, "unsupported_file_type", "当前只支持图片和 PDF 文件")
		return
	}

	fileID := "file_" + newRequestID()
	ext := fileExtension(header.Filename, mimeType)
	now := time.Now()
	objectKey := strings.Join([]string{
		"uploads",
		account.AccountID,
		now.Format("2006"),
		now.Format("01"),
		fileID + ext,
	}, "/")
	hash := sha256.Sum256(buffer)
	storage, err := objectstore.NewMinIOFromConfig(s.configs.GetMap(r.Context()))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "object_store_unavailable", "对象存储配置不可用")
		return
	}
	startedAt := time.Now()
	if _, err := storage.Put(r.Context(), objectKey, bytes.NewReader(buffer), int64(len(buffer)), mimeType); err != nil {
		s.logger.Warn("upload file to minio failed", "error", err)
		writeError(w, http.StatusBadGateway, "object_store_unavailable", "对象存储不可用")
		return
	}
	fileMeta, err := s.files.CreateStoredFile(r.Context(), domain.StoredFileInput{
		FileID:          fileID,
		AccountID:       account.AccountID,
		ObjectKey:       objectKey,
		URL:             "/api/v1/files/" + fileID,
		MimeType:        mimeType,
		SizeBytes:       int64(len(buffer)),
		ContentHash:     hex.EncodeToString(hash[:]),
		StorageProvider: "minio",
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "save_file_failed", "保存文件元数据失败")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"file":               fileMeta,
		"upload_duration_ms": time.Since(startedAt).Milliseconds(),
	})
}

func (s *Server) handleGetFile(w http.ResponseWriter, r *http.Request) {
	account, ok := accountFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "请先登录")
		return
	}
	fileID := strings.TrimPrefix(r.URL.Path, "/api/v1/files/")
	fileID = strings.Trim(fileID, "/")
	if fileID == "" || strings.Contains(fileID, "..") {
		writeError(w, http.StatusBadRequest, "bad_file_id", "文件 ID 不合法")
		return
	}
	fileMeta, ok := s.files.GetStoredFile(r.Context(), fileID)
	if !ok {
		writeError(w, http.StatusNotFound, "file_not_found", "文件不存在")
		return
	}
	if fileMeta.AccountID != "" && fileMeta.AccountID != account.AccountID && account.Role != domain.AccountRoleAdmin {
		writeError(w, http.StatusForbidden, "forbidden", "无权访问该文件")
		return
	}
	storage, err := objectstore.NewMinIOFromConfig(s.configs.GetMap(r.Context()))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "object_store_unavailable", "对象存储配置不可用")
		return
	}
	object, err := storage.Get(r.Context(), fileMeta.ObjectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, "object_not_found", "对象存储文件不存在")
		return
	}
	defer object.Close()
	w.Header().Set("Content-Type", fileMeta.MimeType)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	if fileMeta.SizeBytes > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(fileMeta.SizeBytes, 10))
	}
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, object)
}
