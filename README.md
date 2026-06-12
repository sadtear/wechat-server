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
  trigger_words:
    - "验证码"
    - "登录"
    - "code"
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

- Demo 页的 WeChat Server 地址默认优先使用 `DEMO_BASE_URL` 环境变量；未设置时自动使用当前 Demo 页所在域名（例如本地部署会请求 `http://localhost:端口`），也可在高级工具箱中手动改为其它地址
- 在线测试健康检查、服务状态、验证码换取 OpenID
- 在线修改 WeChat Server 配置（API 密钥、公众号 AppID/AppSecret/Token、验证码长度/有效期、公众号触发词、关注后自动回复等）
- 展示网站登录、注册、绑定已有用户的推荐流程
- 提供 Node.js、PHP、Python、Go 后端接入示例

> 正式业务中请勿把 API 密钥暴露在前端。Demo 页面中的直接调用仅用于部署验收和演示，生产网站应由后端代理调用 `/api/wechat/user`。Demo 页测试台默认优先使用服务端注入的 `DEMO_BASE_URL`；未设置时再使用当前页面 `window.location.origin`。如果你把“高级工具箱 → WeChat Server 地址”手动改成其它域名，才会跨域请求该地址。
>
> 如果 Demo 或 curl 返回 `401` / `未授权访问`，说明请求里的 `Authorization` 密钥与服务端实际加载的 `server.api_token` 不一致，这和公众号 URL、Token、AppID 配置无关。请重点检查运行环境中的 `API_TOKEN` 环境变量；它会覆盖 `config.yaml` 里的 `server.api_token`。
>
> 如果未设置 `DEMO_BASE_URL` 却仍请求旧域名，请先确认访问到的是新镜像/新容器，并强制刷新 `/demo`；当前版本会对 `/demo` 返回 `Cache-Control: no-store`，且页面加载时会用 `DEMO_BASE_URL` 或当前页面域名覆盖浏览器可能恢复的旧表单值。

## API 接口


### 配置管理

```
GET /api/config
POST /api/config
Header: Authorization: {api_token}
```

`POST /api/config` 接收完整配置 JSON，保存到 `CONFIG_PATH` 指向的配置文件（默认 `config.yaml`）并立即更新运行时配置。可通过 `code.trigger_words` 自定义公众号内触发验证码的关键词，例如把默认的 `验证码` 改成 `绑定账号`；可通过 `messages.subscribe_reply` 自定义关注公众号后的自动欢迎回复。

> 如果使用 Docker 挂载配置文件，请确保 `config.yaml` 是可写挂载；如果设置了 `API_TOKEN`、`WECHAT_APPID`、`WECHAT_TOKEN`、`WECHAT_TRIGGER_WORDS`、`WECHAT_SUBSCRIBE_REPLY` 等环境变量，它们会覆盖页面保存的对应配置。保存后请查看接口返回的 `warnings` 和 `data.config`，确认实际生效值。

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

3. **用户发送触发词获取验证码**
   - 用户向公众号发送 `code.trigger_words` 中配置的任意触发词
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
| `DEMO_BASE_URL` | Demo 页默认请求的 WeChat Server 外部访问地址，适合反向代理/CDN 场景，例如 `https://wechat.example.com`；未设置时使用当前页面域名 | - |
| `WECHAT_APPID` | 公众号 AppID（单公众号模式） | - |
| `WECHAT_SECRET` | 公众号 AppSecret | - |
| `WECHAT_TOKEN` | 公众号 Token | - |
| `WECHAT_NAME` | 公众号名称 | - |
| `CODE_LENGTH` | 验证码长度 | 6 |
| `CODE_EXPIRE_MINUTES` | 验证码有效期（分钟） | 5 |
| `WECHAT_TRIGGER_WORDS` | 公众号内触发验证码回复的关键词，支持逗号/分号/换行分隔 | 验证码、登录、code、login、yanzhengma、获取验证码、发送验证码 |
| `WECHAT_SUBSCRIBE_REPLY` | 用户关注公众号后的自动回复内容，支持实际换行或 `\n` 转义换行；会覆盖 `messages.subscribe_reply` | AI80 欢迎语 |

## 许可证

MIT License
