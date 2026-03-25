# LLM Proxy

部署在海外的 LLM API 代理服务，用于转发中国境内对 OpenAI、Claude 等模型的请求。

## 快速开始

### 编译

```bash
make build
```

### 配置

```bash
cp config.yaml.example config.yaml
```

编辑 `config.yaml`，填入你的 API Key：

```yaml
api_keys:
  openrouter: "sk-or-v1-your-key"

auth:
  enabled: true
  users:
    - username: "user1"
      password: "your-password"
```

### 运行

```bash
./llm-proxy -config config.yaml
```

## API 端点

| 端点 | 方法 | 说明 |
|------|------|------|
| `/health` | GET | 健康检查 |
| `/v1/models` | GET | 列出可用模型 |
| `/v1/chat/completions` | POST | 聊天补全 |

## 使用示例

```bash
# 健康检查
curl http://localhost:8080/health

# 列出模型
curl -u user1:password http://localhost:8080/v1/models

# 调用模型
curl -u user1:password http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "anthropic/claude-3.5-sonnet",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

## OpenCode 配置

在 OpenCode 中使用此代理：

```json
{
  "provider": {
    "myproxy": {
      "name": "My LLM Proxy",
      "baseURL": "https://your-server.com/v1",
      "apiKey": "user1:password"
    }
  }
}
```

## License

MIT
