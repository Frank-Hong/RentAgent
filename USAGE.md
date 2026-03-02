# 智能租房 Agent 使用说明

## 功能概览

本 Agent 按《设计目标》与《Agent 对外接口规范和模型调用接口说明》实现：

- **需求解析**：从自然语言中提取预算、地段、户型、通勤、配套等；模糊需求时主动追问。
- **房源处理**：通过 API 自动搜集、筛选、核验房源，支持多平台（链家/安居客/58同城）对比。
- **多维度分析**：对候选房源从通勤、性价比、生活配套、设施等分析，给出优缺点与推荐理由。
- **结果输出**：最多推荐 5 套高匹配度候选房源；**房源查询完成后** response 为规范 JSON：`{"message":"...","houses":["HF_xxx",...]}`。

## 对外接口（规范）

- **Base URL**：`http://localhost:8191`（端口可通过 `AGENT_PORT` 修改）
- **Content-Type**：`application/json`

### POST /api/v1/chat

| 请求字段     | 类型   | 必填 | 说明 |
|--------------|--------|------|------|
| model_ip     | string | 是   | 模型资源接口 IP，端口固定 8888 |
| session_id   | string | 是   | 会话 ID，同一 session 多轮对话需维护上下文 |
| message      | string | 是   | 用户消息 |

| 响应字段     | 类型   | 说明 |
|--------------|--------|------|
| session_id   | string | 会话 ID |
| response     | string | Agent 回复（普通对话为自然语言；房源查询完成为 JSON 字符串） |
| status       | string | 处理状态，如 success / error |
| tool_results | array  | 工具调用结果 [{name, success, output}] |
| timestamp     | int    | 时间戳 |
| duration_ms  | int    | 处理耗时（毫秒） |

**房源查询返回约定**：当完成房源查询后，`response` 必须为合法 JSON 字符串，包含 `message` 与 `houses`（房源 ID 数组），且不能包含自然语言前缀。

## 模型调用（规范）

- 模型地址：`http://{model_ip}:8888`（model_ip 由判题器通过请求参数下发）
- 接口：**POST /v1/chat/completions**
- 请求头：**Session-ID** 必填，取值为当前请求的 `session_id`
- 请求/响应格式与 OpenAI Chat Completion 兼容（messages、tools、stream: false）

## 环境变量

| 变量 | 必填 | 说明 |
|------|------|------|
| `X_USER_ID` | 是 | 用户工号，与比赛平台注册一致，请求头 X-User-ID |
| `AGENT_PORT` | 否 | Agent 监听端口，默认 8191 |
| `RENT_API_BASE_URL` | 否 | 租房仿真 API 地址，默认 `http://7.221.6.201:8080` |
| `LLM_API_KEY` / `LLM_BASE_URL` / `LLM_MODEL` | 否 | 仅本地调测用；大赛时模型由 model_ip 下发 |

## 编译与运行

```bash
go build -o rent-agent ./cmd/rent-agent
export X_USER_ID=你的工号
./rent-agent
```

启动后 Agent 监听 `:8191`。将监听地址配置到大赛平台 Agent 配置中，平台会向 `POST /api/v1/chat` 发送 `model_ip`、`session_id`、`message`。

- **新 session**：首次出现某 `session_id` 时自动调用房源数据重置 `POST /api/houses/init`。
- 同一 `session_id` 的多次请求会保留对话历史，实现多轮上下文。

## 接口与数据

- 房源与地标接口见 **README.md**，工具与参数见 **fake_app_agent_tools.json**。
- 租房/退租/下架须调用对应 API 方为有效。
