package configcenter

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/LYP-leo/xzxg-shop/commerce/internal/domain"
)

type Center interface {
	List(ctx context.Context, includeSecrets bool) []domain.AppConfig
	GetMap(ctx context.Context) map[string]string
	Upsert(ctx context.Context, input domain.AppConfigInput) (domain.AppConfig, error)
}

type MemoryCenter struct {
	mu      sync.RWMutex
	configs map[string]domain.AppConfig
}

func NewMemoryCenter(defaults []domain.AppConfig) *MemoryCenter {
	configs := make(map[string]domain.AppConfig, len(defaults))
	for _, item := range defaults {
		if item.UpdatedAt.IsZero() {
			item.UpdatedAt = time.Now()
		}
		configs[item.ConfigKey] = item
	}
	return &MemoryCenter{configs: configs}
}

func (c *MemoryCenter) List(ctx context.Context, includeSecrets bool) []domain.AppConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	items := make([]domain.AppConfig, 0, len(c.configs))
	for _, item := range c.configs {
		if item.IsSecret && !includeSecrets && item.ConfigValue != "" {
			item.ConfigValue = "******"
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ConfigKey < items[j].ConfigKey })
	return items
}

func (c *MemoryCenter) GetMap(ctx context.Context) map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	values := make(map[string]string, len(c.configs))
	for key, item := range c.configs {
		values[key] = item.ConfigValue
	}
	return values
}

func (c *MemoryCenter) Upsert(ctx context.Context, input domain.AppConfigInput) (domain.AppConfig, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	item := normalizeInput(input)
	c.configs[item.ConfigKey] = item
	return item, nil
}

func normalizeInput(input domain.AppConfigInput) domain.AppConfig {
	item := domain.AppConfig{
		ConfigKey:   strings.TrimSpace(input.ConfigKey),
		ConfigValue: input.ConfigValue,
		ValueType:   input.ValueType,
		Description: input.Description,
		Domain:      input.Domain,
		IsSecret:    input.IsSecret,
		UpdatedAt:   time.Now(),
	}
	if item.Domain == "" {
		item.Domain = DomainForKey(item.ConfigKey)
	}
	if item.ValueType == "" {
		item.ValueType = "string"
	}
	return item
}

func DefaultConfigs() []domain.AppConfig {
	return []domain.AppConfig{
		{ConfigKey: "http.cors.allowed_origins", ConfigValue: "*", ValueType: "string", Description: "允许跨域访问的 Origin，生产环境应配置为明确域名，多个用逗号分隔", Domain: "infra"},
		{ConfigKey: "http.trusted_proxy_cidrs", ConfigValue: "", ValueType: "string", Description: "可信反向代理 CIDR，只有这些来源的 X-Forwarded-For 会被采信", Domain: "infra"},
		{ConfigKey: "http.trust_all_proxies", ConfigValue: "false", ValueType: "bool", Description: "是否信任所有代理头，仅本地调试可开启", Domain: "infra"},
		{ConfigKey: "files.max_upload_bytes", ConfigValue: "10485760", ValueType: "int", Description: "文件上传最大字节数", Domain: "infra"},
		{ConfigKey: "tts.provider", ConfigValue: "xunfei", ValueType: "string", Description: "TTS 供应商：xunfei 或 doubao", Domain: "infra"},
		{ConfigKey: "xunfei.tts.enabled", ConfigValue: "false", ValueType: "bool", Description: "是否启用讯飞在线语音合成代理；生产环境配置密钥后再开启", Domain: "infra"},
		{ConfigKey: "xunfei.tts.app_id", ConfigValue: "", ValueType: "string", Description: "讯飞在线语音合成 AppID；为空时复用 xunfei.app_id", Domain: "infra"},
		{ConfigKey: "xunfei.tts.api_key", ConfigValue: "", ValueType: "string", Description: "讯飞在线语音合成 API Key；为空时复用 xunfei.api_key", Domain: "infra", IsSecret: true},
		{ConfigKey: "xunfei.tts.api_secret", ConfigValue: "", ValueType: "string", Description: "讯飞在线语音合成 API Secret；为空时复用 xunfei.api_secret", Domain: "infra", IsSecret: true},
		{ConfigKey: "xunfei.tts.base_url", ConfigValue: "wss://tts-api.xfyun.cn/v2/tts", ValueType: "string", Description: "讯飞在线语音合成 WebAPI 地址", Domain: "infra"},
		{ConfigKey: "xunfei.tts.voice", ConfigValue: "xiaoyan", ValueType: "string", Description: "默认 TTS 发音人", Domain: "infra"},
		{ConfigKey: "xunfei.tts.speed", ConfigValue: "50", ValueType: "int", Description: "TTS 语速，0-100", Domain: "infra"},
		{ConfigKey: "xunfei.tts.volume", ConfigValue: "50", ValueType: "int", Description: "TTS 音量，0-100", Domain: "infra"},
		{ConfigKey: "xunfei.tts.pitch", ConfigValue: "50", ValueType: "int", Description: "TTS 音高，0-100", Domain: "infra"},
		{ConfigKey: "xunfei.tts.timeout_seconds", ConfigValue: "20", ValueType: "int", Description: "TTS 单次合成超时时间", Domain: "infra"},
		{ConfigKey: "xunfei.tts.max_runes", ConfigValue: "800", ValueType: "int", Description: "TTS 单次合成最大字符数", Domain: "infra"},
		{ConfigKey: "doubao.tts.enabled", ConfigValue: "false", ValueType: "bool", Description: "是否启用豆包/火山语音合成代理；需同时设置 tts.provider=doubao", Domain: "infra"},
		{ConfigKey: "doubao.tts.app_id", ConfigValue: "", ValueType: "string", Description: "豆包/火山语音合成 AppID", Domain: "infra"},
		{ConfigKey: "doubao.tts.api_key", ConfigValue: "", ValueType: "string", Description: "豆包/火山语音合成 Access Token/API Key", Domain: "infra", IsSecret: true},
		{ConfigKey: "doubao.tts.base_url", ConfigValue: "https://openspeech.bytedance.com/api/v1/tts", ValueType: "string", Description: "豆包/火山语音合成 HTTP API 地址", Domain: "infra"},
		{ConfigKey: "doubao.tts.cluster", ConfigValue: "volcano_tts", ValueType: "string", Description: "豆包/火山语音合成集群标识", Domain: "infra"},
		{ConfigKey: "doubao.tts.voice", ConfigValue: "BV700_streaming", ValueType: "string", Description: "豆包/火山语音合成默认音色 voice_type", Domain: "infra"},
		{ConfigKey: "doubao.tts.encoding", ConfigValue: "mp3", ValueType: "string", Description: "豆包/火山语音合成音频编码，默认 mp3", Domain: "infra"},
		{ConfigKey: "doubao.tts.uid", ConfigValue: "xzxg-shop", ValueType: "string", Description: "豆包/火山语音合成用户标识", Domain: "infra"},
		{ConfigKey: "doubao.tts.speed_ratio", ConfigValue: "1.0", ValueType: "number", Description: "豆包/火山语音合成语速倍率", Domain: "infra"},
		{ConfigKey: "doubao.tts.volume_ratio", ConfigValue: "1.0", ValueType: "number", Description: "豆包/火山语音合成音量倍率", Domain: "infra"},
		{ConfigKey: "doubao.tts.pitch_ratio", ConfigValue: "1.0", ValueType: "number", Description: "豆包/火山语音合成音高倍率", Domain: "infra"},
		{ConfigKey: "doubao.tts.timeout_seconds", ConfigValue: "20", ValueType: "int", Description: "豆包/火山语音合成单次超时时间", Domain: "infra"},
		{ConfigKey: "doubao.tts.max_runes", ConfigValue: "800", ValueType: "int", Description: "豆包/火山语音合成单次最大字符数", Domain: "infra"},
		{ConfigKey: "minio.endpoint", ConfigValue: "127.0.0.1:9100", ValueType: "string", Description: "MinIO/S3 对象存储地址", Domain: "infra"},
		{ConfigKey: "minio.access_key", ConfigValue: "minioadmin", ValueType: "string", Description: "MinIO Access Key，生产环境必须改为独立账号", Domain: "infra", IsSecret: true},
		{ConfigKey: "minio.secret_key", ConfigValue: "minioadmin", ValueType: "string", Description: "MinIO Secret Key，生产环境必须改为强密钥", Domain: "infra", IsSecret: true},
		{ConfigKey: "minio.bucket", ConfigValue: "xzxg-commerce-assets", ValueType: "string", Description: "对象存储 bucket", Domain: "infra"},
		{ConfigKey: "minio.use_ssl", ConfigValue: "false", ValueType: "bool", Description: "对象存储是否启用 HTTPS", Domain: "infra"},
	}
}

func DomainForKey(key string) string { return "infra" }
