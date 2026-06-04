package main

import (
	"embed"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/seefs001/wechat-server/config"
	"github.com/seefs001/wechat-server/handler"
)

// demoFS 内嵌接入测试页，确保 Docker 运行时无需额外挂载静态文件。
//
//go:embed demo/demo.html
var demoFS embed.FS

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 检查配置
	if len(cfg.Accounts) == 0 {
		log.Println("警告: 未配置任何公众号账户")
	} else {
		log.Printf("已加载 %d 个公众号配置", len(cfg.Accounts))
		for _, acc := range cfg.Accounts {
			log.Printf("  - %s (%s)", acc.Name, acc.AppID)
		}
	}

	if cfg.Server.APIToken == "" {
		log.Println("警告: 未配置 API Token，API 接口将不受保护")
	}

	// 设置 Gin 模式
	gin.SetMode(gin.ReleaseMode)

	// 创建路由
	r := gin.Default()

	// 接入测试 Demo 页面
	r.GET("/demo", func(c *gin.Context) {
		data, err := demoFS.ReadFile("demo/demo.html")
		if err != nil {
			c.String(500, "Demo 页面加载失败")
			return
		}
		c.Data(200, "text/html; charset=utf-8", data)
	})

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	// 微信消息接口（支持多公众号）
	r.GET("/wechat/:app_id", handler.WechatVerify)
	r.POST("/wechat/:app_id", handler.WechatMessage)

	// 兼容单公众号模式（使用默认第一个公众号）
	r.GET("/wechat", handler.WechatVerifyDefault)
	r.POST("/wechat", handler.WechatMessageDefault)

	// API 接口（供 Yi-API 调用）
	api := r.Group("/api")
	{
		api.GET("/wechat/user", handler.AuthMiddleware(), handler.GetUser)
		api.GET("/wechat/stats", handler.AuthMiddleware(), handler.GetStats)
	}

	// 启动服务
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("WeChat Server 启动在 %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("启动服务失败: %v", err)
	}
}
