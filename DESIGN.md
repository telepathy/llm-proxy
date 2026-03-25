# LLM API 代理服务设计文档

## 1. 项目概述
一个部署在海外服务器的代理服务，用于转发来自中国境内对 LLM API（OpenAI、Claude、Gemini 等）的请求。该服务兼容 OpenAI API 格式，支持通过 OpenRouter 或直接访问各个模型提供商，解决中国境内访问海外 LLM API 的网络限制问题。

## 2. 核心需求
- **解决访问限制**：中国境内无法直接访问某些 LLM API，通过海外代理绕过
- **API 兼容**：提供 OpenAI 兼容的 API 接口，可直接作为 OpenCode 的 backend 使用
- **灵活路由**：支持配置不同模型走不同的后端（OpenRouter / 直连提供商）
- **认证**：HTTP 基础认证保护代理服务
- **部署**：海外服务器直接部署

## 3. 架构设计

### 3.1 整体架构
```
┌─────────────────────────────────────────────────────────────┐
│                        中国境内                              │
│  ┌─────────────┐                                            │
│  │   OpenCode   │                                            │
│  │   客户端     │                                            │
│  └──────┬──────┘                                            │
│         │ OpenAI 兼容 API                                    │
└─────────┼───────────────────────────────────────────────────┘
          │
          ▼
┌─────────────────────────────────────────────────────────────┐
│                     海外服务器                               │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              opencode-proxy                          │   │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────────────┐  │   │
│  │  │  认证层   │→│  路由层   │→│  请求转发/转换层   │  │   │
│  │  └──────────┘  └──────────┘  └──────────────────┘  │   │
│  └─────────────────────────────────────────────────────┘   │
│         │                        │                         │
│         ▼                        ▼                         │
│  ┌──────────────┐        ┌──────────────────┐              │
│  │   OpenRouter  │        │  直连 LLM 提供商  │              │
│  │  (统一网关)   │        │  OpenAI/Claude/  │              │
│  │              │        │  Gemini/etc      │              │
│  └──────────────┘        └──────────────────┘              │
└─────────────────────────────────────────────────────────────┘
```

### 3.2 核心组件
1. **认证中间件**：HTTP 基础认证，保护代理服务
2. **路由器**：根据配置的规则决定请求走 OpenRouter 还是直连提供商
3. **请求转换层**：如果直连提供商，需要转换为对应提供商的 API 格式
4. **反向代理**：使用 `net/http/httputil.ReverseProxy` 转发请求
5. **响应处理**：统一响应格式，确保 OpenAI 兼容

## 4. 技术选型
- **语言**：Go
- **配置格式**：YAML
- **依赖**：尽量使用标准库

## 5. 配置文件设计 (`config.yaml`)

```yaml
# 服务配置
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: 120s   # LLM 响应可能较慢
  write_timeout: 120s

# 认证配置（代理自身的认证）
auth:
  enabled: true
  users:
    - username: "user1"
      password: "password1"  # 生产环境应使用 bcrypt 哈希

# API Keys 配置（访问上游服务的认证）
api_keys:
  openrouter: "sk-or-v1-xxxxx"
  openai: "sk-xxxxx"
  anthropic: "sk-ant-xxxxx"

# 默认后端配置
default_backend: "openrouter"

# 路由配置 - 模型到后端的映射
routes:
  # 通过 OpenRouter 访问的模型
  - model_pattern: "*"  # 默认所有模型走 OpenRouter
    backend: "openrouter"
    
  # 直连 OpenAI 的模型（如果需要更低延迟或绕过 OpenRouter 限制）
  - model_pattern: "gpt-4o"
    backend: "openai-direct"
    
  # 直连 Anthropic 的模型
  - model_pattern: "claude-3*"
    backend: "anthropic-direct"

# 后端定义
backends:
  # OpenRouter 统一网关
  openrouter:
    type: "openrouter"
    base_url: "https://openrouter.ai/api/v1"
    api_key: "${api_keys.openrouter}"
    # OpenRouter 特定 headers
    extra_headers:
      HTTP-Referer: "https://your-site.com"
      X-Title: "My Proxy"
    
  # 直连 OpenAI
  openai-direct:
    type: "openai"
    base_url: "https://api.openai.com/v1"
    api_key: "${api_keys.openai}"
    
  # 直连 Anthropic
  anthropic-direct:
    type: "anthropic"
    base_url: "https://api.anthropic.com"
    api_key: "${api_keys.anthropic}"
    
  # 自定义后端（任何 OpenAI 兼容 API）
  custom:
    type: "openai-compatible"
    base_url: "https://your-custom-llm.com/v1"
    api_key: "your-key"

# 日志配置
logging:
  level: "info"
  format: "json"
  file: "/var/log/llm-proxy.log"
  # 记录请求详情（注意脱敏）
  log_requests: true
  log_responses: false  # 响应可能很大
```

## 6. API 兼容性

### 6.1 OpenAI 兼容端点
代理实现以下 OpenAI API 端点：

| 端点 | 方法 | 说明 |
|------|------|------|
| `/v1/chat/completions` | POST | 聊天补全（核心端点） |
| `/v1/completions` | POST | 文本补全（旧版） |
| `/v1/models` | GET | 列出可用模型 |
| `/v1/embeddings` | POST | 文本嵌入 |

### 6.2 请求/响应格式
完全兼容 OpenAI API 格式：

```json
// 请求
POST /v1/chat/completions
Authorization: Bearer your-proxy-api-key
Content-Type: application/json

{
  "model": "anthropic/claude-3.5-sonnet",
  "messages": [
    {"role": "user", "content": "Hello!"}
  ],
  "stream": true
}

// 响应（流式）
data: {"id":"chatcmpl-xxx","choices":[{"delta":{"content":"Hi"}}]}

data: [DONE]
```

### 6.3 模型名称映射
支持多种模型名称格式：

| 输入格式 | 路由目标 |
|----------|----------|
| `gpt-4o` | OpenRouter 或 OpenAI 直连（根据配置） |
| `openai/gpt-4o` | OpenRouter（带提供商前缀） |
| `anthropic/claude-3.5-sonnet` | OpenRouter 或 Anthropic 直连 |
| `claude-3-5-sonnet-20241022` | Anthropic 直连（原始名称） |

## 7. 部署方案

### 7.1 编译和部署
```bash
# 编译
go build -o llm-proxy cmd/proxy/main.go

# 运行
./llm-proxy -config config.yaml
```

### 7.2 systemd 服务
```ini
[Unit]
Description=LLM API Proxy Service
After=network.target

[Service]
Type=simple
User=llm-proxy
Group=llm-proxy
ExecStart=/usr/local/bin/llm-proxy -config /etc/llm-proxy/config.yaml
Restart=always
RestartSec=5
Environment=GIN_MODE=release

[Install]
WantedBy=multi-user.target
```

### 7.3 使用示例

#### OpenCode 配置
```json
{
  "provider": {
    "myproxy": {
      "npm": "@ai-sdk/openai-compatible",
      "name": "My LLM Proxy",
      "options": {
        "baseURL": "https://your-overseas-server.com/v1",
        "apiKey": "your-proxy-api-key"
      },
      "models": {
        "gpt-4o": { "name": "GPT-4o" },
        "claude-3.5-sonnet": { "name": "Claude 3.5 Sonnet" }
      }
    }
  }
}
```

#### curl 测试
```bash
# 通过代理调用 Claude
curl -X POST https://your-server.com/v1/chat/completions \
  -H "Authorization: Basic $(echo -n 'user:pass' | base64)" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "anthropic/claude-3.5-sonnet",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

## 8. 安全考虑
1. **密码哈希**：配置文件中的密码使用 bcrypt
2. **HTTPS**：生产环境必须启用 TLS
3. **API Key 保护**：日志中脱敏，不记录完整 API Key
4. **速率限制**：防止滥用，可按用户/IP 限制
5. **请求验证**：验证请求格式，拒绝恶意请求

## 9. 监控和日志
1. **健康检查**：`GET /health`
2. **指标**：`GET /metrics`（Prometheus 格式）
   - 请求总数、延迟、错误率
   - 按模型/后端分组的统计
3. **结构化日志**：JSON 格式，包含请求 ID、耗时、模型、状态码

## 10. 项目结构
```
llm-proxy/
├── cmd/
│   └── proxy/
│       └── main.go              # 主程序入口
├── internal/
│   ├── config/
│   │   ├── config.go            # 配置结构定义
│   │   └── loader.go            # 配置加载
│   ├── auth/
│   │   ├── basic.go             # 基础认证
│   │   └── middleware.go        # 认证中间件
│   ├── router/
│   │   ├── router.go            # 路由逻辑
│   │   └── pattern.go           # 模型名模式匹配
│   ├── backend/
│   │   ├── interface.go         # 后端接口定义
│   │   ├── openrouter.go        # OpenRouter 后端
│   │   ├── openai.go            # OpenAI 直连后端
│   │   ├── anthropic.go         # Anthropic 直连后端
│   │   └── factory.go           # 后端工厂
│   ├── proxy/
│   │   ├── handler.go           # 请求处理器
│   │   ├── stream.go            # 流式响应处理
│   │   └── transform.go         # 请求/响应转换
│   └── logging/
│       └── logger.go            # 日志工具
├── config.yaml.example          # 配置示例
├── go.mod
├── go.sum
├── Makefile
├── README.md
└── DESIGN.md
```

## 11. 后续步骤
1. 确认设计方案
2. 实现基本代理框架
3. 添加 OpenRouter 后端支持
4. 添加直连提供商支持
5. 测试和部署