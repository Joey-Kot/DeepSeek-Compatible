# DeepSeek 多协议仿真器

[English](README.md) | 中文

这是一个面向 DeepSeek 的协议仿真器。它对外提供 DeepSeek Chat Completions、OpenAI Chat Completions、OpenAI Responses、Anthropic Messages 和 Gemini Generate Content 等兼容 API，并在 DeepSeek Chat Completions 之上尽可能高保真地仿真这些协议的请求与响应语义。

本地运行时通过命令行参数进行配置，容器部署时通过环境变量进行配置。

## 配置与使用

本地二进制运行时，服务通过命令行参数配置。
可以拉取项目后自行编译运行或直接从 Release 下载：

```bash
git clone https://github.com/Joey-Kot/deepseek-compatible.git
cd deepseek-compatible
go build -trimpath -ldflags="-s -w" -o deepseek-compatible ./cmd/server
```

```bash
./deepseek-compatible \
  --listen :8080 \
  --api-token sk-local-test \
  --deepseek-api-key sk-your-deepseek-key \
  --deepseek-base-url https://api.deepseek.com \
  --deepseek-model deepseek-v4-pro \
  --deepseek-models deepseek-v4-pro \
  --deepseek-http-timeout 120 \
  --verify-ssl=true \
  --debug-log-body=false
```

容器部署时使用环境变量。可以先从 `docker.env.example` 复制一份配置：

```bash
cp docker.env.example docker.env
```

编辑 `docker.env` 后，可以直接拉取线上镜像部署：

```bash
docker run -itd \
  --name deepseek-compatible \
  -p 8080:8080 \
  --env-file docker.env \
  --restart always \
  ghcr.io/joey-kot/deepseek-compatible:latest
```

也可以拉取项目后自行构建镜像部署：

```bash
git clone https://github.com/Joey-Kot/deepseek-compatible.git
cd deepseek-compatible
docker build -t deepseek-compatible:latest .
```

```bash
docker run -itd \
  --name deepseek-compatible \
  -p 8080:8080 \
  --env-file docker.env \
  --restart always \
  deepseek-compatible:latest
```

容器环境变量说明：

| 环境变量 | 对应参数 |
| --- | --- |
| `LISTEN` | `--listen` |
| `API_TOKEN` | `--api-token` |
| `DEEPSEEK_API_KEY` | `--deepseek-api-key` |
| `DEEPSEEK_BASE_URL` | `--deepseek-base-url` |
| `DEEPSEEK_MODEL` | `--deepseek-model` |
| `DEEPSEEK_MODELS` | `--deepseek-models` |
| `DEEPSEEK_HTTP_TIMEOUT` | `--deepseek-http-timeout` |
| `DEEPSEEK_MAX_IDLE_CONNS` | `--deepseek-max-idle-conns` |
| `DEEPSEEK_MAX_IDLE_CONNS_PER_HOST` | `--deepseek-max-idle-conns-per-host` |
| `DEEPSEEK_MAX_CONNS_PER_HOST` | `--deepseek-max-conns-per-host` |
| `READ_HEADER_TIMEOUT` | `--read-header-timeout` |
| `IDLE_TIMEOUT` | `--idle-timeout` |
| `VERIFY_SSL` | `--verify-ssl` |
| `DEBUG_LOG_BODY` | `--debug-log-body` |

参数说明：

| 参数 | 描述 |
| --- | --- |
| `--listen` | 本地 HTTP 监听地址，默认 `:8080`。 |
| `--api-token` | 访问本兼容后端所需的本地 token，支持用逗号配置多个 token；OpenAI 风格请求使用 `Authorization: Bearer`，Anthropic 风格请求使用 `x-api-key`，Gemini 风格请求使用 `x-goog-api-key`。 |
| `--deepseek-api-key` | DeepSeek 上游 API key。 |
| `--deepseek-base-url` | DeepSeek 上游 base URL；不填写或填写为空字符串时使用默认值 `https://api.deepseek.com`，也可以填写 `http://` 或 `https://`，并且可以直接指向 `/chat/completions`。 |
| `--deepseek-model` | 默认转发到 DeepSeek 的模型 ID，默认 `deepseek-v4-pro`。 |
| `--deepseek-models` | `/v1/models` 对外暴露的模型 ID 列表，多个模型用逗号分隔；如果未包含默认模型，会自动把默认模型放到列表前面。 |
| `--deepseek-http-timeout` | DeepSeek 上游 HTTP 请求超时时间，单位为秒，默认 `120`。 |
| `--deepseek-max-idle-conns` | 上游 HTTP 空闲连接复用池总上限，默认 `200`。 |
| `--deepseek-max-idle-conns-per-host` | 每个上游 host 保留的空闲连接上限，默认 `100`。 |
| `--deepseek-max-conns-per-host` | 每个上游 host 的并发连接上限，默认 `0` 表示不限制。 |
| `--read-header-timeout` | 本地 HTTP 读取请求头超时时间，单位为秒，默认 `10`。 |
| `--idle-timeout` | 本地 HTTP 空闲连接超时时间，单位为秒，默认 `120`。 |
| `--verify-ssl` | 是否校验 DeepSeek 上游 HTTPS 证书，默认 `true`；只有在可信代理或临时证书异常场景下才建议设为 `false`。 |
| `--debug-log-body` | 是否输出经过脱敏的本地请求/响应 body 和 DeepSeek 上游请求/响应 body，默认 `false`；API key、token、password、secret 等字段会被替换为 `[REDACTED]`，日志长度也会被限制。 |

完整参数可以参考 `args.example`。

## 兼容端点

### DeepSeek Chat Completions

| 端点 | 描述 |
| --- | --- |
| `POST /chat/completions` | 创建 DeepSeek Chat Completions，请求和响应参数均与 DeepSeek 官方 API 一一对应。 |

### OpenAI Chat Completions

| 端点 | 描述 |
| --- | --- |
| `POST /v1/chat/completions` | 创建 Chat Completions，并转发到 DeepSeek Chat Completions。 |
| `GET /v1/chat/completions` | 列出本地保存的 Chat Completions。 |
| `GET /v1/chat/completions/{completion_id}` | 读取本地保存的单个 Chat Completion。 |
| `POST /v1/chat/completions/{completion_id}` | 更新本地保存的 Chat Completion 元数据。 |
| `DELETE /v1/chat/completions/{completion_id}` | 删除本地保存的 Chat Completion。 |
| `GET /v1/chat/completions/{completion_id}/messages` | 列出本地保存的 Chat Completion 消息。 |

### OpenAI Responses

| 端点 | 描述 |
| --- | --- |
| `POST /v1/responses` | 创建 Responses，并转发到 DeepSeek Chat Completions。 |
| `GET /v1/responses/{response_id}` | 读取本地保存的单个 Response。 |
| `DELETE /v1/responses/{response_id}` | 删除本地保存的 Response。 |
| `GET /v1/responses/{response_id}/input_items` | 列出本地保存的 Response 输入项。 |
| `POST /v1/responses/{response_id}/cancel` | 按本地状态语义取消 Response。 |
| `POST /v1/responses/input_tokens` | 使用内置 DeepSeek 官方 tokenizer 在本地统计输入 token。 |
| `POST /v1/responses/compact` | 使用 DeepSeek 进行尽力而为的上下文压缩总结。 |

NOTICE：针对 Codex CLI 的 MCP namespace 工具调用，Responses 适配器会在转发给 DeepSeek 前将 namespace 和工具名展开成 DeepSeek 可接受的 function 名称，并在返回时尽量还原为 Responses 工具调用结构。

### OpenAI Conversations

| 端点 | 描述 |
| --- | --- |
| `POST /v1/conversations` | 创建本地 Conversation。 |
| `GET /v1/conversations/{conversation_id}` | 读取本地 Conversation。 |
| `POST /v1/conversations/{conversation_id}` | 追加或更新本地 Conversation。 |
| `DELETE /v1/conversations/{conversation_id}` | 删除本地 Conversation。 |

### Anthropic Messages

| 端点 | 描述 |
| --- | --- |
| `POST /v1/messages` | 创建 Anthropic Messages 响应，并转发到 DeepSeek Chat Completions。 |
| `POST /v1/messages/count_tokens` | 使用内置 DeepSeek 官方 tokenizer 在本地统计 Anthropic Messages token。 |

### Gemini Generate Content

| 端点 | 描述 |
| --- | --- |
| `POST /v1beta/models/{model}:generateContent` | 创建 Gemini Generate Content 响应，并转发到 DeepSeek Chat Completions。 |
| `POST /v1beta/models/{model}:streamGenerateContent` | 创建 Gemini 流式 Generate Content 响应。 |
| `POST /v1beta/models/{model}:countTokens` | 使用内置 DeepSeek 官方 tokenizer 在本地统计 Gemini v1beta token。 |
| `POST /v1/models/{model}:generateContent` | 创建 Gemini v1 Generate Content 响应，并转发到 DeepSeek Chat Completions。 |
| `POST /v1/models/{model}:streamGenerateContent` | 创建 Gemini v1 流式 Generate Content 响应。 |
| `POST /v1/models/{model}:countTokens` | 使用内置 DeepSeek 官方 tokenizer 在本地统计 Gemini v1 token。 |

### 通用端点

| 端点 | 描述 |
| --- | --- |
| `GET /v1/models` | 返回当前暴露给兼容客户端的模型列表。 |
| `GET /health` | 健康检查端点。 |

## 参数映射

### DeepSeek Chat Completions

| DeepSeek Chat Completions | DeepSeek Chat Completions |
| --- | --- |
| 所有请求参数 | 原样转发到 DeepSeek 上游。 |
| 非流式响应 | 原样返回 DeepSeek 上游响应。 |
| 流式响应 | 原样转发 DeepSeek 上游 SSE chunk。 |

`POST /chat/completions` 只进行本地鉴权、调试日志脱敏和上游错误处理，不做参数转换。

### OpenAI Chat Completions

| OpenAI Chat Completions | DeepSeek Chat Completions |
| --- | --- |
| `model` | `model` |
| `messages` | `messages` |
| `developer` role | `system` role |
| `max_completion_tokens` / `max_tokens` | `max_tokens` |
| `temperature` | `temperature` |
| `top_p` | `top_p` |
| `stop` | `stop` |
| `presence_penalty` | `presence_penalty` |
| `frequency_penalty` | `frequency_penalty` |
| `logprobs` | `logprobs` |
| `top_logprobs` | `top_logprobs` |
| `n` | `n` |
| `seed` | `seed` |
| `stream` | DeepSeek 流式响应转为 Chat Completions SSE chunk。 |
| `stream_options.include_usage` | `stream_options.include_usage` |
| `tools` function tools | `tools` |
| 已废弃的 `functions` | `tools` |
| `tool_choice` / 已废弃的 `function_call` | `tool_choice` |
| `tool_choice.type=allowed_tools` | 尽力过滤可用工具，并映射为 `auto` / `required`。 |
| `response_format.type=json_object` | `response_format={"type":"json_object"}` |
| `response_format.type=json_schema` | 尽力开启 JSON mode，并将 schema 写入提示。 |
| `reasoning_effort` | DeepSeek `reasoning_effort`，其中 `low` / `medium` 会转为 `high`，`xhigh` 会转为 `max`。 |
| `thinking` | DeepSeek `thinking` |

OpenAI Chat Completions 官方响应结构不包含 reasoning summary；因此 DeepSeek 返回的 `reasoning_content` 会作为 DeepSeek 扩展字段保留并透传，不转换成 OpenAI Responses 的 `reasoning.summary[].summary_text` 结构。

当请求设置 `store=true` 时，Chat Completions 会保存在本地内存中。读取、更新 metadata、删除、按 `metadata[key]` 列表过滤，以及消息列表都属于本地兼容状态；DeepSeek 本身不保存这些对象，服务重启后本地状态会丢失。

### OpenAI Responses

| OpenAI Responses | DeepSeek Chat Completions |
| --- | --- |
| `model` | `model` |
| `input` string | `messages: [{role: "user", content: input}]` |
| `input` message items | `messages` |
| `instructions` | 前置 `system` message |
| `max_output_tokens` | `max_tokens` |
| `temperature` | `temperature` |
| `top_p` | `top_p` |
| `stop` | `stop` |
| `tools` function tools | Chat Completions `tools` |
| `tools` namespace function/custom tools，包括 Codex CLI MCP tools | 展平为 Chat Completions function tools |
| `tool_choice` for function/custom tools | Chat Completions `tool_choice` |
| `text.format.type=json_object` | `response_format={"type":"json_object"}` |
| `text.format.type=json_schema` | 尽力开启 JSON mode，并将 schema 写入提示。 |
| `reasoning.effort` | DeepSeek `thinking` 和 `reasoning_effort` |
| `stream=true` | DeepSeek 流式响应转为 Responses SSE events。 |

DeepSeek 返回的 `reasoning_content` 会映射为 Responses 输出中的 `reasoning.summary[].summary_text`；流式响应中会通过 reasoning summary 相关 SSE 事件逐段输出。Responses namespace tools，包括 Codex CLI 本地 MCP namespace 的工具形态，会在发送到 DeepSeek 前展平为 function tools，并在返回时尽量还原为 `namespace` / `name` 结构。OpenAI 托管工具，例如 web search、file search、code interpreter、image generation、computer use、remote MCP connectors、moderation 和后台队列，并不存在于 DeepSeek Chat Completions 中，因此不会转发。

### Anthropic Messages

| Anthropic Messages | DeepSeek Chat Completions |
| --- | --- |
| `model` | `model` |
| `messages[].role=user/assistant` | `messages[].role=user/assistant` |
| 顶层 `system` | 前置 `system` message |
| text content blocks | message `content` text |
| `thinking` content blocks | assistant `reasoning_content` |
| `tool_use` blocks | assistant `tool_calls` |
| `tool_result` blocks | `tool` messages |
| `max_tokens` | `max_tokens` |
| `temperature` | `temperature` |
| `top_p` | `top_p` |
| `stop_sequences` | `stop` |
| `tools[].input_schema` | function tool `parameters` |
| `tool_choice.type=auto/any/none/tool` | `auto` / `required` / `none` / named function |
| `thinking.type=enabled/disabled` | DeepSeek `thinking` |
| `output_config.format=json_schema` | 尽力开启 JSON mode，并将 schema 写入提示。 |
| `stream=true` | Anthropic Messages SSE events |

Anthropic 的 image、document、search-result 和 server-tool blocks 会尽量转换为文本描述，作为 DeepSeek 上下文发送。DeepSeek 返回的 tool calls 会转换为 Anthropic `tool_use` content blocks。token counting 使用内置 DeepSeek 官方 tokenizer 在本地完成。

### Gemini Generate Content

| Gemini Generate Content | DeepSeek Chat Completions |
| --- | --- |
| path `{model}` | `model` |
| `contents[].role=user/model` | `user` / `assistant` messages |
| `contents[].parts[].text` | message `content` text |
| `systemInstruction.parts[].text` | 前置 `system` message |
| `functionCall` parts | assistant `tool_calls` |
| `functionResponse` parts | `tool` messages |
| `generationConfig.maxOutputTokens` | `max_tokens` |
| `generationConfig.temperature` | `temperature` |
| `generationConfig.topP` | `top_p` |
| `generationConfig.stopSequences` | `stop` |
| `generationConfig.responseMimeType=application/json` | `response_format={"type":"json_object"}` |
| `generationConfig.responseSchema` | 尽力开启 JSON mode，并将 schema 写入提示。 |
| `generationConfig.thinkingConfig` | DeepSeek `thinking` / `reasoning_effort` |
| `tools[].functionDeclarations` | function tools |
| `toolConfig.functionCallingConfig` | `tool_choice`，并尽力过滤可用工具。 |
| `:streamGenerateContent` | Gemini SSE chunks |

Gemini 多模态输入 parts 和内置工具会尽量转换为文本上下文。通过 `functionDeclarations` 定义的函数会映射为 DeepSeek function tools；DeepSeek 返回的 tool calls 会转换为 Gemini `functionCall` parts。

## 请求示例

健康检查：

```bash
curl http://localhost:8080/health
```

创建 DeepSeek Chat Completion：

```bash
curl http://localhost:8080/chat/completions \
  -H "Authorization: Bearer sk-local-test" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deepseek-v4-pro",
    "messages": [
      {"role": "system", "content": "You are a helpful assistant."},
      {"role": "user", "content": "Hello!"}
    ],
    "reasoning_effort": "high"
  }'
```

创建 OpenAI Responses：

```bash
curl http://localhost:8080/v1/responses \
  -H "Authorization: Bearer sk-local-test" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deepseek-v4-pro",
    "instructions": "You are a helpful assistant.",
    "input": "Hello!",
    "reasoning": {"effort": "high"}
  }'
```

创建 OpenAI Chat Completion：

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-local-test" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deepseek-v4-pro",
    "messages": [
      {"role": "system", "content": "You are a helpful assistant."},
      {"role": "user", "content": "Hello!"}
    ],
    "reasoning_effort": "high"
  }'
```

创建 Anthropic Message：

```bash
curl http://localhost:8080/v1/messages \
  -H "x-api-key: sk-local-test" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deepseek-v4-pro",
    "max_tokens": 128,
    "system": "You are a helpful assistant.",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

创建 Gemini Generate Content：

```bash
curl http://localhost:8080/v1beta/models/gemini-3.5-flash:generateContent \
  -H "x-goog-api-key: sk-local-test" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [{
      "role": "user",
      "parts": [{"text": "Hello!"}]
    }]
  }'
```

创建 OpenAI Responses 流式响应：

```bash
curl http://localhost:8080/v1/responses \
  -H "Authorization: Bearer sk-local-test" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deepseek-v4-pro",
    "input": "Write one haiku.",
    "stream": true
  }'
```

使用 OpenAI Responses function tool：

```bash
curl http://localhost:8080/v1/responses \
  -H "Authorization: Bearer sk-local-test" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deepseek-v4-pro",
    "input": "What is the weather in New York?",
    "tools": [{
      "type": "function",
      "name": "get_weather",
      "description": "Get weather by city.",
      "parameters": {
        "type": "object",
        "properties": {"city": {"type": "string"}},
        "required": ["city"]
      }
    }],
    "tool_choice": "auto"
  }'
```

## 兼容性说明

本后端会在内存中保存 Responses 和 Conversations 状态，用于支持 `previous_response_id`、`conversation`、读取、删除和输入项列表等兼容能力。如果没有接入外部存储，服务重启后这些本地状态会丢失。

DeepSeek Chat Completions 暂未提供服务端 token counting 端点，因此本后端内置 DeepSeek 官方 tokenizer，在本地为 OpenAI Responses、Anthropic Messages 和 Gemini Generate Content 的 token 计算端点提供统计能力。

`POST /v1/responses/{id}/cancel` 只能按本地状态语义标记尚未完成的 Response。普通请求是同步完成的，DeepSeek Chat Completions 也不提供 OpenAI 后台任务式执行能力。

DeepSeek 可能返回 `reasoning_content`。本后端会在目标 API 存在结构化 reasoning 形态时进行映射：OpenAI Responses 映射为 `reasoning.summary[].summary_text`，Anthropic 映射为 `thinking` blocks，Gemini 映射为 `thought` parts。OpenAI Chat Completions 官方响应结构不包含 reasoning summary，因此 Chat Completions 会保留并透传 DeepSeek 的 `reasoning_content` 扩展字段。

## 许可证

本项目基于 GNU General Public License v3.0 or later（GPLv3+）授权。详情请查看 [LICENSE](LICENSE)。
