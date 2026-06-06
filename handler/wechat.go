package handler

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/seefs001/wechat-server/config"
	"github.com/seefs001/wechat-server/store"
	"github.com/seefs001/wechat-server/wechat"
)

// WechatVerify 微信服务器验证（指定公众号）
func WechatVerify(c *gin.Context) {
	appID := c.Param("app_id")
	account := config.GetAccountByAppID(appID)
	if account == nil {
		c.String(http.StatusNotFound, "公众号未配置")
		return
	}

	verifyWithToken(c, account.Token)
}

// WechatVerifyDefault 微信服务器验证（默认公众号）
func WechatVerifyDefault(c *gin.Context) {
	cfg := config.Get()
	if len(cfg.Accounts) == 0 {
		c.String(http.StatusNotFound, "未配置任何公众号")
		return
	}

	verifyWithToken(c, cfg.Accounts[0].Token)
}

func verifyWithToken(c *gin.Context, token string) {
	signature := c.Query("signature")
	timestamp := c.Query("timestamp")
	nonce := c.Query("nonce")
	echostr := c.Query("echostr")

	if wechat.VerifySignature(token, signature, timestamp, nonce) {
		c.String(http.StatusOK, echostr)
	} else {
		c.String(http.StatusForbidden, "验证失败")
	}
}

// WechatMessage 处理微信消息（指定公众号）
func WechatMessage(c *gin.Context) {
	appID := c.Param("app_id")
	account := config.GetAccountByAppID(appID)
	if account == nil {
		c.String(http.StatusNotFound, "公众号未配置")
		return
	}

	handleMessage(c, account)
}

// WechatMessageDefault 处理微信消息（默认公众号）
func WechatMessageDefault(c *gin.Context) {
	cfg := config.Get()
	if len(cfg.Accounts) == 0 {
		c.String(http.StatusNotFound, "未配置任何公众号")
		return
	}

	handleMessage(c, &cfg.Accounts[0])
}

func handleMessage(c *gin.Context, account *config.WechatAccount) {
	// 读取请求体
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("读取请求体失败: %v", err)
		c.String(http.StatusBadRequest, "")
		return
	}

	// 解析消息
	msg, err := wechat.ParseMessage(body)
	if err != nil {
		log.Printf("解析消息失败: %v", err)
		c.String(http.StatusBadRequest, "")
		return
	}

	log.Printf("[%s] 收到消息: type=%s, from=%s, content=%s",
		account.Name, msg.MsgType, msg.GetOpenID(), msg.Content)

	// 转发消息到配置的转发器
	// 如果转发器返回有效的 XML 响应，优先使用该响应
	forwardResponse := ForwardMessage(account, msg, body)
	if forwardResponse != nil {
		log.Printf("[%s] 使用转发器响应", account.Name)
		c.Data(http.StatusOK, "application/xml", forwardResponse)
		return
	}

	// 没有转发器响应，使用本地默认处理
	log.Printf("[%s] 使用本地默认响应", account.Name)

	// 处理消息
	var replyContent string
	switch {
	case msg.IsSubscribeEvent():
		// 关注事件
		replyContent = "欢迎关注 AI80！\n\n" +
			"我们提供 AI API 聚合平台，一个 Key 用遍 Claude、GPT、Gemini 等主流模型。\n\n" +
			"<a href=\"https://code.ai80.vip\">Code80 — AI API 平台（主站）</a>\n" +
			"<a href=\"https://chat.ai80.vip\">AI 对话助手（在线体验）</a>\n\n" +
			"回复【教程】获取接入指南\n" +
			"回复【客服】联系人工客服\n\n" +
			"平台支持按量计费、团队共享、API Key 自助管理。\n" +
			"推广合作详情回复【合作】"

	case msg.IsUnsubscribeEvent():
		// 取消关注事件，不回复
		c.String(http.StatusOK, "")
		return

	case msg.IsTextMessage():
		// 文本消息，检查是否是请求验证码
		if config.IsVerificationCodeRequest(msg.Content) {
			// 用户请求验证码
			openID := msg.GetOpenID()
			code := store.GetStore().GenerateCode(openID, account.AppID)
			replyContent = fmt.Sprintf("您的登录验证码是：%s\n\n验证码有效期 %d 分钟，请尽快使用。",
				code, config.Get().Code.ExpireMinutes)
		} else {
			// 其他文本消息，不自动生成验证码
			c.String(http.StatusOK, "success")
			return
		}

	default:
		// 其他消息类型（图片、语音等），不回复
		c.String(http.StatusOK, "success")
		return
	}

	// 构建回复
	reply := wechat.NewTextReply(msg.FromUserName, msg.ToUserName, replyContent)
	replyXML, err := reply.ToXML()
	if err != nil {
		log.Printf("生成回复失败: %v", err)
		c.String(http.StatusInternalServerError, "")
		return
	}

	c.Data(http.StatusOK, "application/xml", replyXML)
}
