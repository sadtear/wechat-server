package config

import (
	"os"
	"strconv"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// Config 全局配置
type Config struct {
	Server   ServerConfig    `yaml:"server" json:"server"`
	Accounts []WechatAccount `yaml:"accounts" json:"accounts"`
	Code     CodeConfig      `yaml:"code" json:"code"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port     int    `yaml:"port" json:"port"`
	APIToken string `yaml:"api_token" json:"api_token"`
}

// WechatAccount 微信公众号账户配置
type WechatAccount struct {
	AppID      string      `yaml:"app_id" json:"app_id"`
	AppSecret  string      `yaml:"app_secret" json:"app_secret"`
	Token      string      `yaml:"token" json:"token"`
	Name       string      `yaml:"name" json:"name"`
	Forwarders []Forwarder `yaml:"forwarders" json:"forwarders"` // 消息转发配置
}

// Forwarder 消息转发器配置
type Forwarder struct {
	Name     string   `yaml:"name" json:"name"`         // 转发器名称
	URL      string   `yaml:"url" json:"url"`           // 转发目标URL
	Priority int      `yaml:"priority" json:"priority"` // 优先级（数字越小优先级越高）
	Events   []string `yaml:"events" json:"events"`     // 要转发的事件类型，"*" 表示全部
	Timeout  int      `yaml:"timeout" json:"timeout"`   // 超时时间（毫秒），默认5000
}

// CodeConfig 验证码配置
type CodeConfig struct {
	Length        int      `yaml:"length" json:"length"`
	ExpireMinutes int      `yaml:"expire_minutes" json:"expire_minutes"`
	TriggerWords  []string `yaml:"trigger_words" json:"trigger_words"` // 用户在公众号内触发验证码回复的关键词
}

var (
	cfg        *Config
	once       sync.Once
	cfgMu      sync.RWMutex
	configPath string
)

var defaultTriggerWords = []string{
	"验证码",
	"登录",
	"code",
	"login",
	"yanzhengma",
	"获取验证码",
	"发送验证码",
}

// Load 加载配置
func Load() (*Config, error) {
	var err error
	once.Do(func() {
		loaded := defaultConfig()

		// 尝试从配置文件加载
		configPath = getEnv("CONFIG_PATH", "config.yaml")
		if data, readErr := os.ReadFile(configPath); readErr == nil {
			if parseErr := yaml.Unmarshal(data, loaded); parseErr != nil {
				err = parseErr
				return
			}
		}

		applyEnvOverrides(loaded)
		normalizeConfig(loaded)

		cfgMu.Lock()
		cfg = loaded
		cfgMu.Unlock()
	})

	cfgMu.RLock()
	current := cfg
	cfgMu.RUnlock()
	if current == nil {
		return defaultConfig(), err
	}
	return current, err
}

// Get 获取配置（必须先调用 Load）
func Get() *Config {
	if cfg == nil {
		Load()
	}
	cfgMu.RLock()
	defer cfgMu.RUnlock()
	if cfg == nil {
		return defaultConfig()
	}
	return cfg
}

// Save 保存配置到 config.yaml，并立即更新运行时配置。
func Save(newCfg *Config) error {
	if newCfg == nil {
		newCfg = defaultConfig()
	}
	if configPath == "" {
		configPath = getEnv("CONFIG_PATH", "config.yaml")
	}

	normalizeConfig(newCfg)
	data, err := yaml.Marshal(newCfg)
	if err != nil {
		return err
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return err
	}

	cfgMu.Lock()
	cfg = newCfg
	cfgMu.Unlock()
	return nil
}

// GetConfigPath 返回当前配置文件路径。
func GetConfigPath() string {
	if configPath == "" {
		return getEnv("CONFIG_PATH", "config.yaml")
	}
	return configPath
}

// GetAccountByAppID 根据 AppID 获取公众号配置
func GetAccountByAppID(appID string) *WechatAccount {
	if cfg == nil {
		Load()
	}
	cfgMu.RLock()
	defer cfgMu.RUnlock()
	for _, acc := range cfg.Accounts {
		if acc.AppID == appID {
			account := acc
			return &account
		}
	}
	return nil
}

// GetAccountByToken 根据 Token 获取公众号配置
func GetAccountByToken(token string) *WechatAccount {
	if cfg == nil {
		Load()
	}
	cfgMu.RLock()
	defer cfgMu.RUnlock()
	for _, acc := range cfg.Accounts {
		if acc.Token == token {
			account := acc
			return &account
		}
	}
	return nil
}

// IsVerificationCodeRequest 判断文本内容是否命中验证码触发词。
func IsVerificationCodeRequest(content string) bool {
	content = strings.TrimSpace(strings.ToLower(content))
	if content == "" {
		return false
	}

	if cfg == nil {
		Load()
	}
	cfgMu.RLock()
	words := append([]string(nil), cfg.Code.TriggerWords...)
	cfgMu.RUnlock()

	for _, word := range words {
		keyword := strings.TrimSpace(strings.ToLower(word))
		if keyword != "" && strings.Contains(content, keyword) {
			return true
		}
	}
	return false
}

func defaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:     3000,
			APIToken: "",
		},
		Code: CodeConfig{
			Length:        6,
			ExpireMinutes: 5,
			TriggerWords:  append([]string(nil), defaultTriggerWords...),
		},
	}
}

func applyEnvOverrides(target *Config) {
	if port := os.Getenv("PORT"); port != "" {
		if p, parseErr := strconv.Atoi(port); parseErr == nil {
			target.Server.Port = p
		}
	}
	if token := os.Getenv("API_TOKEN"); token != "" {
		target.Server.APIToken = token
	}
	if length := os.Getenv("CODE_LENGTH"); length != "" {
		if l, parseErr := strconv.Atoi(length); parseErr == nil {
			target.Code.Length = l
		}
	}
	if expire := os.Getenv("CODE_EXPIRE_MINUTES"); expire != "" {
		if e, parseErr := strconv.Atoi(expire); parseErr == nil {
			target.Code.ExpireMinutes = e
		}
	}
	if triggerWords := os.Getenv("WECHAT_TRIGGER_WORDS"); triggerWords != "" {
		target.Code.TriggerWords = splitWords(triggerWords)
	}

	// 从环境变量加载单个公众号配置（向后兼容）
	if appID := os.Getenv("WECHAT_APPID"); appID != "" {
		account := WechatAccount{
			AppID:     appID,
			AppSecret: os.Getenv("WECHAT_SECRET"),
			Token:     os.Getenv("WECHAT_TOKEN"),
			Name:      os.Getenv("WECHAT_NAME"),
		}
		// 检查是否已存在
		exists := false
		for i, acc := range target.Accounts {
			if acc.AppID == appID {
				target.Accounts[i] = account
				exists = true
				break
			}
		}
		if !exists {
			target.Accounts = append(target.Accounts, account)
		}
	}
}

func normalizeConfig(target *Config) {
	if target.Server.Port <= 0 {
		target.Server.Port = 3000
	}
	if target.Code.Length <= 0 {
		target.Code.Length = 6
	}
	if target.Code.ExpireMinutes <= 0 {
		target.Code.ExpireMinutes = 5
	}
	target.Code.TriggerWords = normalizeWords(target.Code.TriggerWords)
	if len(target.Code.TriggerWords) == 0 {
		target.Code.TriggerWords = append([]string(nil), defaultTriggerWords...)
	}
}

func normalizeWords(words []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(words))
	for _, word := range words {
		trimmed := strings.TrimSpace(word)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, trimmed)
	}
	return result
}

func splitWords(value string) []string {
	return normalizeWords(strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == '，' || r == '\n' || r == ';' || r == '；'
	}))
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
