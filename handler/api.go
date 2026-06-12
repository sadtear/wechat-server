package handler

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/seefs001/wechat-server/config"
	"github.com/seefs001/wechat-server/store"
)

// AuthMiddleware API Token 认证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg := config.Get()

		// 如果未配置 token，跳过验证
		if cfg.Server.APIToken == "" {
			c.Next()
			return
		}

		// 从 Authorization 头获取 token
		auth := c.GetHeader("Authorization")
		token := strings.TrimPrefix(auth, "Bearer ")
		token = strings.TrimSpace(token)

		if token != cfg.Server.APIToken {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "未授权访问",
				"data":    "",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetConfig 获取当前运行时配置，供 Demo 页面编辑使用
func GetConfig(c *gin.Context) {
	cfg := config.Get()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"config":      cfg,
			"config_path": config.GetConfigPath(),
			"warnings":    configWarnings(),
		},
	})
}

// UpdateConfig 更新配置文件并立即应用到运行时
func UpdateConfig(c *gin.Context) {
	var newCfg config.Config
	if err := c.ShouldBindJSON(&newCfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "配置格式错误: " + err.Error(),
			"data":    "",
		})
		return
	}

	if err := config.Save(&newCfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "保存配置失败: " + err.Error(),
			"data":    "",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "配置已保存并生效",
		"data": gin.H{
			"config":      config.Get(),
			"config_path": config.GetConfigPath(),
			"warnings":    configWarnings(),
		},
	})
}

func configWarnings() []string {
	warnings := make([]string, 0)
	if os.Getenv("API_TOKEN") != "" {
		warnings = append(warnings, "当前设置了 API_TOKEN 环境变量；它会覆盖页面保存的 server.api_token")
	}
	if os.Getenv("WECHAT_APPID") != "" || os.Getenv("WECHAT_SECRET") != "" || os.Getenv("WECHAT_TOKEN") != "" || os.Getenv("WECHAT_NAME") != "" {
		warnings = append(warnings, "当前设置了 WECHAT_APPID/WECHAT_SECRET/WECHAT_TOKEN/WECHAT_NAME 单公众号环境变量；它们会覆盖页面保存的同类公众号配置")
	}
	if os.Getenv("CODE_LENGTH") != "" || os.Getenv("CODE_EXPIRE_MINUTES") != "" {
		warnings = append(warnings, "当前设置了 CODE_LENGTH 或 CODE_EXPIRE_MINUTES 环境变量；它们会覆盖页面保存的验证码长度或有效期")
	}
	if os.Getenv("WECHAT_TRIGGER_WORDS") != "" {
		warnings = append(warnings, "当前设置了 WECHAT_TRIGGER_WORDS 环境变量；它会覆盖页面保存的 code.trigger_words")
	}
	if os.Getenv("WECHAT_SUBSCRIBE_REPLY") != "" {
		warnings = append(warnings, "当前设置了 WECHAT_SUBSCRIBE_REPLY 环境变量；它会覆盖页面保存的 messages.subscribe_reply")
	}
	return warnings
}

// GetUser 验证验证码并返回用户信息（Yi-API 调用）
func GetUser(c *gin.Context) {
	code := c.Query("code")
	appID := c.Query("app_id") // 可选，用于多公众号模式

	if code == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "验证码不能为空",
			"data":    "",
		})
		return
	}

	// 验证验证码
	openID := store.GetStore().VerifyCode(code, appID)
	if openID == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "验证码错误或已过期",
			"data":    "",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    openID,
	})
}

// GetStats 获取服务状态统计
func GetStats(c *gin.Context) {
	codeCount, userCount := store.GetStore().Stats()
	cfg := config.Get()

	accounts := make([]gin.H, 0, len(cfg.Accounts))
	for _, acc := range cfg.Accounts {
		accounts = append(accounts, gin.H{
			"app_id": acc.AppID,
			"name":   acc.Name,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"active_codes": codeCount,
			"active_users": userCount,
			"accounts":     accounts,
		},
	})
}
