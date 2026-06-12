package config

import (
	"os"
	"strings"
	"testing"
)

func TestIsVerificationCodeRequestUsesConfiguredTriggerWords(t *testing.T) {
	cfgMu.Lock()
	original := cfg
	cfg = &Config{
		Server: ServerConfig{Port: 3000},
		Code: CodeConfig{
			Length:        6,
			ExpireMinutes: 5,
			TriggerWords:  []string{"绑定账号", "LoginNow"},
		},
	}
	cfgMu.Unlock()
	defer func() {
		cfgMu.Lock()
		cfg = original
		cfgMu.Unlock()
	}()

	if !IsVerificationCodeRequest("请帮我绑定账号") {
		t.Fatal("expected custom Chinese trigger word to generate a verification code")
	}
	if !IsVerificationCodeRequest("loginnow") {
		t.Fatal("expected trigger matching to be case-insensitive")
	}
	if IsVerificationCodeRequest("验证码") {
		t.Fatal("did not expect old default trigger word to match after runtime config changed")
	}
}

func TestNormalizeConfigDefaultsTriggerWords(t *testing.T) {
	cfg := &Config{}
	normalizeConfig(cfg)

	if cfg.Server.Port != 3000 {
		t.Fatalf("expected default port 3000, got %d", cfg.Server.Port)
	}
	if cfg.Code.Length != 6 {
		t.Fatalf("expected default code length 6, got %d", cfg.Code.Length)
	}
	if cfg.Code.ExpireMinutes != 5 {
		t.Fatalf("expected default expire minutes 5, got %d", cfg.Code.ExpireMinutes)
	}
	if len(cfg.Code.TriggerWords) == 0 || cfg.Code.TriggerWords[0] != "验证码" {
		t.Fatalf("expected default trigger words, got %#v", cfg.Code.TriggerWords)
	}
	if !strings.Contains(cfg.Messages.SubscribeReply, "欢迎关注 AI80") {
		t.Fatalf("expected default subscribe reply, got %q", cfg.Messages.SubscribeReply)
	}
}

func TestSavePersistsFileAndRuntimeHonorsEnvOverrides(t *testing.T) {
	t.Setenv("API_TOKEN", "env-token")
	t.Setenv("WECHAT_TRIGGER_WORDS", "env-word")
	t.Setenv("WECHAT_SUBSCRIBE_REPLY", `env welcome\nsecond line`)

	cfgMu.Lock()
	originalCfg := cfg
	originalPath := configPath
	cfg = nil
	configPath = t.TempDir() + "/config.yaml"
	cfgMu.Unlock()
	defer func() {
		cfgMu.Lock()
		cfg = originalCfg
		configPath = originalPath
		cfgMu.Unlock()
	}()

	err := Save(&Config{
		Server: ServerConfig{Port: 3000, APIToken: "posted-token"},
		Accounts: []WechatAccount{{
			AppID: "wx-posted",
			Token: "posted-wechat-token",
			Name:  "posted",
		}},
		Code: CodeConfig{
			Length:        6,
			ExpireMinutes: 5,
			TriggerWords:  []string{"posted-word"},
		},
		Messages: MessageConfig{SubscribeReply: "posted welcome"},
	})
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	saved, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read saved config: %v", err)
	}
	if !strings.Contains(string(saved), "posted-token") || !strings.Contains(string(saved), "posted-word") || !strings.Contains(string(saved), "posted welcome") {
		t.Fatalf("expected posted values to be persisted, got:\n%s", string(saved))
	}

	effective := Get()
	if effective.Server.APIToken != "env-token" {
		t.Fatalf("expected runtime API token to honor env override, got %q", effective.Server.APIToken)
	}
	if !IsVerificationCodeRequest("env-word") {
		t.Fatal("expected env trigger word to be effective at runtime")
	}
	if IsVerificationCodeRequest("posted-word") {
		t.Fatal("did not expect posted trigger word to override WECHAT_TRIGGER_WORDS at runtime")
	}
	if effective.Messages.SubscribeReply != "env welcome\nsecond line" {
		t.Fatalf("expected runtime subscribe reply to honor WECHAT_SUBSCRIBE_REPLY, got %q", effective.Messages.SubscribeReply)
	}
}
