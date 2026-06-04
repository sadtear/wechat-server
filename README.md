# WeChat Server

基于微信公众号的验证码登录服务，可集成到任意需要微信登录功能的应用系统。

## 功能特性

- 支持多公众号管理
- 验证码自动生成与过期
- RESTful API 接口，易于集成
- Docker 一键部署
- 最小依赖，轻量高效

## 快速开始

### Docker 部署（推荐）

1. 创建配置文件：

```bash
cp config.example.yaml config.yaml
# 编辑 config.yaml，填入你的公众号配置
```

2. 启动服务：

```bash
docker-compose up -d
```

### 手动部署

```bash
# 下载依赖
go mod download

# 运行
go run main.go

# 或编译后运行
go build -o wechat-server
./wechat-server
```

## 配置说明

### 多公众号模式（推荐）

使用 `config.yaml` 配置多个公众号：

```yaml
server:
  port: 3000
  api_token: "your-api-token"

accounts:
  - app_id: "wx1234567890"
    app_secret: "secret1"
    token: "token1"
    name: "公众号A"

  - app_id: "wx0987654321"
    app_secret: "secret2"
    token: "token2"
    name: "公众号B"

code:
  length: 6
  expire_minutes: 5
```

### 单公众号模式

使用环境变量配置：

```bash
export PORT=3000
export API_TOKEN=your-api-token
export WECHAT_APPID=wx1234567890
export WECHAT_SECRET=your-app-secret
export WECHAT_TOKEN=your-wechat-token
export WECHAT_NAME=我的公众号
```

## 微信公众号配置

1. 登录 [微信公众平台](https://mp.weixin.qq.com/)
2. 进入 **设置与开发 → 基本配置**
3. 配置服务器：
   - **URL**: `https://your-domain.com/wechat/{app_id}`
   - **Token**: 与配置文件中的 `token` 一致
   - **EncodingAESKey**: 随机生成（可选）
   - **消息加解密方式**: 明文模式

### 多公众号 URL 配置

每个公众号配置独立的服务器地址：

| 公众号 | 服务器 URL |
|--------|------------|
| 公众号A | `https://wechat.example.com/wechat/wx1234567890` |
| 公众号B | `https://wechat.example.com/wechat/wx0987654321` |


### 接入测试 Demo 页面

部署完成后可以直接访问内置 Demo 页面，用作接口测试台和给接入方的示例文档：

```
https://your-domain.com/demo
```

Demo 页面支持：

- 预填 WeChat Server 地址、API 密钥、公众号 AppID 等接入参数
- 在线测试健康检查、服务状态、验证码换取 OpenID
- 展示网站登录、注册、绑定已有用户的推荐流程
- 提供 Node.js、PHP、Python、Go 后端接入示例

> 正式业务中请勿把 API 密钥暴露在前端。Demo 页面中的直接调用仅用于部署验收和演示，生产网站应由后端代理调用 `/api/wechat/user`。

## API 接口

### 验证用户

```
GET /api/wechat/user?code={验证码}&app_id={公众号AppID}
Header: Authorization: {api_token}

成功响应:
{
    "success": true,
    "message": "",
    "data": "用户OpenID"
}

错误响应:
{
    "success": false,
    "message": "验证码错误或已过期",
    "data": ""
}
```

### 服务状态

```
GET /api/wechat/stats
Header: Authorization: {api_token}

响应:
{
    "success": true,
    "data": {
        "active_codes": 10,
        "active_users": 5,
        "accounts": [
            {"app_id": "wx123", "name": "公众号A"}
        ]
    }
}
```

### 健康检查

```
GET /health

响应:
{
    "status": "ok"
}
```

## 集成到你的应用

在你的应用系统中集成微信登录功能：

1. **配置 WeChat Server 地址和凭证**
   - 服务器地址: `https://your-domain.com`
   - API 访问凭证: 配置文件中的 `api_token`

2. **前端展示公众号二维码**
   - 引导用户扫码关注公众号

3. **用户发送消息获取验证码**
   - 用户向公众号发送任意消息
   - 公众号自动回复 6 位验证码（有效期 5 分钟）

4. **验证用户身份**
   - 调用 `/api/wechat/user` 接口验证验证码
   - 获取用户的微信 OpenID
   - 完成登录或注册流程

## 工作流程

```
┌─────────┐    扫码关注     ┌─────────────┐
│  用户   │ ──────────────→ │ 微信公众号  │
└─────────┘                 └─────────────┘
     │                            │
     │ 输入验证码                  │ 发送验证码
     ↓                            ↓
┌─────────┐   验证 code    ┌─────────────┐
│ 你的应用 │ ──────────────→│WeChat Server│
└─────────┘                └─────────────┘
     │                            │
     │←── 返回 OpenID ────────────┘
     │
     ↓
  登录/注册成功
```

## 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `PORT` | 服务端口 | 3000 |
| `API_TOKEN` | API 访问凭证 | - |
| `CONFIG_PATH` | 配置文件路径 | config.yaml |
| `WECHAT_APPID` | 公众号 AppID（单公众号模式） | - |
| `WECHAT_SECRET` | 公众号 AppSecret | - |
| `WECHAT_TOKEN` | 公众号 Token | - |
| `WECHAT_NAME` | 公众号名称 | - |
| `CODE_LENGTH` | 验证码长度 | 6 |
| `CODE_EXPIRE_MINUTES` | 验证码有效期（分钟） | 5 |

## 许可证

MIT License
