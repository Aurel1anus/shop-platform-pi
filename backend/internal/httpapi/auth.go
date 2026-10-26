package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/LYP-leo/xzxg-shop/commerce/internal/auth"
)

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, 400, "bad_request", "请求 JSON 不合法")
		return
	}
	if strings.TrimSpace(input.Username) == "" || input.Password == "" {
		writeError(w, 400, "empty_credential", "账号和密码不能为空")
		return
	}
	account, token, err := s.auth.Login(r.Context(), input.Username, input.Password)
	if errors.Is(err, auth.ErrInvalidCredentials) {
		writeError(w, 401, "invalid_credential", err.Error())
		return
	}
	if err != nil {
		writeError(w, 500, "login_failed", "登录失败")
		return
	}
	writeJSON(w, 200, map[string]any{"account": account, "token": token})
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username    string `json:"username"`
		Password    string `json:"password"`
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, 400, "bad_request", "请求 JSON 不合法")
		return
	}
	account, token, err := s.auth.Register(r.Context(), input.Username, input.Password, input.DisplayName)
	if errors.Is(err, auth.ErrInvalidRegistration) {
		writeError(w, 400, "invalid_register_input", err.Error())
		return
	}
	if errors.Is(err, auth.ErrUsernameExists) {
		writeError(w, 409, "username_exists", err.Error())
		return
	}
	if err != nil {
		writeError(w, 500, "register_failed", "注册失败")
		return
	}
	writeJSON(w, 201, map[string]any{"account": account, "token": token})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	account, ok := accountFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "请先登录")
		return
	}
	writeJSON(w, http.StatusOK, account)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r)
	if token != "" {
		s.auth.DeleteAuthToken(r.Context(), token)
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleAccountProfile(w http.ResponseWriter, r *http.Request) {
	account, ok := accountFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "请先登录")
		return
	}
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, account)
		return
	}
	var request struct {
		DisplayName string `json:"display_name"`
		Nickname    string `json:"nickname"`
		AvatarURL   string `json:"avatar_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "请求 JSON 不合法")
		return
	}
	displayName := strings.TrimSpace(request.DisplayName)
	if displayName == "" {
		displayName = strings.TrimSpace(request.Nickname)
	}
	if displayName == "" && strings.TrimSpace(request.AvatarURL) == "" {
		writeError(w, http.StatusBadRequest, "empty_profile", "昵称或头像不能为空")
		return
	}
	if displayName != "" && len([]rune(displayName)) > 24 {
		writeError(w, http.StatusBadRequest, "bad_display_name", "昵称最多 24 个字符")
		return
	}
	updated, ok := s.auth.UpdateAccountProfile(r.Context(), account.AccountID, displayName, request.AvatarURL)
	if !ok {
		writeError(w, http.StatusNotFound, "account_not_found", "账号不存在")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleAccountContact(w http.ResponseWriter, r *http.Request) {
	account, ok := accountFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "请先登录")
		return
	}
	var request struct {
		Phone string `json:"phone"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "请求 JSON 不合法")
		return
	}
	phone := strings.TrimSpace(request.Phone)
	email := strings.TrimSpace(request.Email)
	if len([]rune(phone)) > 32 || len([]rune(email)) > 128 {
		writeError(w, http.StatusBadRequest, "bad_contact", "手机号或邮箱过长")
		return
	}
	if email != "" && (!strings.Contains(email, "@") || strings.Contains(email, " ")) {
		writeError(w, http.StatusBadRequest, "bad_email", "邮箱格式不正确")
		return
	}
	updated, ok := s.auth.UpdateAccountContact(r.Context(), account.AccountID, phone, email)
	if !ok {
		writeError(w, http.StatusNotFound, "account_not_found", "账号不存在")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	account, ok := accountFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "请先登录")
		return
	}
	if !s.auth.DeleteAccount(r.Context(), account.AccountID) {
		writeError(w, http.StatusInternalServerError, "delete_account_failed", "删除账号失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleUploadAvatar(w http.ResponseWriter, r *http.Request) {
	account, ok := accountFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "请先登录")
		return
	}
	if err := r.ParseMultipartForm(3 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "bad_upload", "头像上传请求不合法")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing_file", "请选择头像文件")
		return
	}
	defer file.Close()
	if header.Size > 2<<20 {
		writeError(w, http.StatusBadRequest, "file_too_large", "头像不能超过 2MB")
		return
	}
	buffer, err := io.ReadAll(io.LimitReader(file, 2<<20+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "read_file_failed", "读取头像失败")
		return
	}
	mimeType := http.DetectContentType(buffer)
	ext := ".jpg"
	switch mimeType {
	case "image/jpeg":
		ext = ".jpg"
	case "image/png":
		ext = ".png"
	case "image/webp":
		ext = ".webp"
	default:
		writeError(w, http.StatusBadRequest, "bad_image_type", "头像仅支持 JPG、PNG 或 WebP")
		return
	}
	root := avatarUploadRoot()
	if err := os.MkdirAll(root, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, "mkdir_failed", "保存头像失败")
		return
	}
	filename := account.AccountID + "_" + time.Now().Format("20060102150405") + ext
	if err := os.WriteFile(filepath.Join(root, filename), buffer, 0o644); err != nil {
		writeError(w, http.StatusInternalServerError, "save_file_failed", "保存头像失败")
		return
	}
	url := "/api/v1/uploads/avatar/" + filename
	writeJSON(w, http.StatusOK, map[string]any{
		"url":       url,
		"mime_type": mimeType,
		"size":      len(buffer),
	})
}

func (s *Server) handleListAdminAccounts(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	page, pageSize := readPagination(r)
	items, total := s.auth.ListAccountsPage(r.Context(), page, pageSize)
	writeJSON(w, http.StatusOK, pagedPayload(items, page, pageSize, total))
}

func (s *Server) handleUpdateAdminAccount(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	accountID := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/accounts/")
	var request struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "请求 JSON 不合法")
		return
	}
	if !allowedAccountStatus(request.Status) {
		writeError(w, http.StatusBadRequest, "bad_status", "账号状态不合法")
		return
	}
	account, ok := s.auth.UpdateAccountStatus(r.Context(), accountID, request.Status)
	if !ok {
		writeError(w, http.StatusNotFound, "account_not_found", "账号不存在")
		return
	}
	writeJSON(w, http.StatusOK, account)
}
