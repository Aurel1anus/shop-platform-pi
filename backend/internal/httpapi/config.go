package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/LYP-leo/xzxg-shop/commerce/internal/domain"
)

func (s *Server) handleListAdminConfigs(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	page, pageSize := readPagination(r)
	all := s.configs.List(r.Context(), false)
	itemsForPage := make([]domain.AppConfig, 0, len(all))
	for _, item := range all {

		itemsForPage = append(itemsForPage, item)
	}
	items, total := paginateList(itemsForPage, page, pageSize)
	writeJSON(w, http.StatusOK, pagedPayload(items, page, pageSize, total))
}

func (s *Server) handleUpdateAdminConfig(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	configKey := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/configs/")
	if configKey == "" || strings.Contains(configKey, "/") {
		writeError(w, http.StatusNotFound, "not_found", "接口不存在")
		return
	}
	var request struct {
		Value       string `json:"value"`
		ValueType   string `json:"value_type"`
		Description string `json:"description"`
		IsSecret    *bool  `json:"is_secret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "请求 JSON 不合法")
		return
	}
	isSecret := strings.Contains(strings.ToLower(configKey), "key") || strings.Contains(strings.ToLower(configKey), "secret")
	if request.IsSecret != nil {
		isSecret = *request.IsSecret
	}
	if isSecret && request.Value == "******" {
		for _, existing := range s.configs.List(r.Context(), true) {
			if existing.ConfigKey == configKey {
				request.Value = existing.ConfigValue
				break
			}
		}
	}
	config, err := s.configs.Upsert(r.Context(), domain.AppConfigInput{
		ConfigKey:   configKey,
		ConfigValue: request.Value,
		ValueType:   request.ValueType,
		Description: request.Description,
		IsSecret:    isSecret,
	})
	if err != nil {
		s.logger.Error("update app config failed", "error", err, "config_key", configKey)
		writeError(w, http.StatusInternalServerError, "update_config_failed", "更新配置失败")
		return
	}
	if config.IsSecret && config.ConfigValue != "" {
		config.ConfigValue = "******"
	}
	writeJSON(w, http.StatusOK, config)
}
