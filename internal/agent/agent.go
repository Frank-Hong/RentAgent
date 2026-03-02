package agent

import (
	"encoding/json"
	"fmt"
	"regexp"

	"rentagent/internal/api"
)

// ToolResult 单次工具调用结果，对应对外接口的 tool_results 元素
type ToolResult struct {
	Name    string `json:"name"`
	Success bool   `json:"success"`
	Output  string `json:"output"`
}

const (
	systemPrompt = `你是智能租房助手，帮助用户在北京高效找到合适房源。

核心职责：
1. 需求解析：从用户自然语言中精准提取租金预算、地段、户型、通勤（如到西二旗时间）、生活配套、设施、租赁周期等；若需求模糊（如“通勤方便、性价比高”），主动追问关键信息直至明确。
2. 房源处理：通过工具自动搜集、筛选、核验房源，可多平台（链家/安居客/58同城）对比；按适配度、租金、地段等分类。
3. 多维度分析：对候选房源从通勤距离、租金性价比、生活配套（商超、地铁、公园）、房屋设施、隐性信息（噪音、采光等）进行分析，给出客观优缺点。
4. 结果输出：最终最多推荐 5 套高匹配度候选房源，每条包含推荐理由与关键信息，便于用户决策。

规则：
- 房源相关接口必须带 X-User-ID（已由系统处理）。
- 每个新会话开始时会自动调用房源数据重置，保证数据为初始状态。
- 用户确定租房时，必须调用 rent_house 接口完成租房，仅对话中说“已租”无效；退租、下架同理，须调用对应接口。
- 近地铁指到最近地铁站直线距离 800 米以内；地标附近查房需先查地标得到 landmark_id，再调用 get_houses_nearby。
- 输出简洁、条理清晰，便于用户快速理解与决策。

重要：当你完成房源推荐、需要返回房源列表时，你的最后一条回复必须是且仅是合法的 JSON 字符串，不要加任何前后自然语言。格式为：{"message": "给用户的简短说明", "houses": ["HF_xxx", "HF_yyy", ...]}，其中 houses 为最多 5 个房源 ID 的数组。若未找到合适房源，可返回 {"message": "说明", "houses": []}。`
)

// Agent 智能租房 Agent
type Agent struct {
	llm    *LLMClient
	client *api.Client
	tools  []ToolDef
}

// New 创建 Agent
func New(llm *LLMClient, client *api.Client) *Agent {
	return &Agent{
		llm:    llm,
		client: client,
		tools:  ToolDefinitions(),
	}
}

// Run 运行一轮对话：传入当前消息历史与用户最新输入，返回助手回复；内部会执行工具调用并循环直到无 tool_calls
func (a *Agent) Run(messages []ChatMessage, userInput string) (string, error) {
	msgs := make([]ChatMessage, 0, len(messages)+1)
	msgs = append(msgs, messages...)
	msgs = append(msgs, ChatMessage{Role: "user", Content: userInput})

	// 若尚无 system，则插入
	hasSystem := false
	for _, m := range msgs {
		if m.Role == "system" {
			hasSystem = true
			break
		}
	}
	if !hasSystem {
		inner := make([]ChatMessage, 0, len(msgs)+1)
		inner = append(inner, ChatMessage{Role: "system", Content: systemPrompt})
		inner = append(inner, msgs...)
		msgs = inner
	}

	maxRounds := 15
	for round := 0; round < maxRounds; round++ {
		msg, err := a.llm.Chat(msgs, a.tools)
		if err != nil {
			return "", err
		}
		msgs = append(msgs, msg)

		if len(msg.ToolCalls) == 0 {
			return msg.Content, nil
		}

		// 执行每个 tool call，并追加 tool 结果消息
		for _, tc := range msg.ToolCalls {
			name := tc.Function.Name
			args := tc.Function.Arguments
			result, err := ExecuteTool(a.client, name, args)
			if err != nil {
				result = "错误: " + err.Error()
			}
			msgs = append(msgs, ChatMessage{
				Role:       "tool",
				ToolCallID: tc.ID,
				Content:    result,
			})
		}
	}
	return "", fmt.Errorf("超过最大工具调用轮数 %d", maxRounds)
}

// RunWithHistory 与 Run 相同，但返回更新后的消息历史（便于多轮对话）
func (a *Agent) RunWithHistory(messages []ChatMessage, userInput string) (reply string, newMessages []ChatMessage, err error) {
	msgs := make([]ChatMessage, 0, len(messages)+1)
	msgs = append(msgs, messages...)
	msgs = append(msgs, ChatMessage{Role: "user", Content: userInput})
	hasSystem := false
	for _, m := range msgs {
		if m.Role == "system" {
			hasSystem = true
			break
		}
	}
	if !hasSystem {
		inner := make([]ChatMessage, 0, len(msgs)+1)
		inner = append(inner, ChatMessage{Role: "system", Content: systemPrompt})
		inner = append(inner, msgs...)
		msgs = inner
	}

	maxRounds := 15
	for round := 0; round < maxRounds; round++ {
		msg, err := a.llm.Chat(msgs, a.tools)
		if err != nil {
			return "", nil, err
		}
		msgs = append(msgs, msg)
		if len(msg.ToolCalls) == 0 {
			return msg.Content, msgs, nil
		}
		for _, tc := range msg.ToolCalls {
			result, execErr := ExecuteTool(a.client, tc.Function.Name, tc.Function.Arguments)
			if execErr != nil {
				result = "错误: " + execErr.Error()
			}
			msgs = append(msgs, ChatMessage{
				Role:       "tool",
				ToolCallID: tc.ID,
				Content:    result,
			})
		}
	}
	return "", nil, fmt.Errorf("超过最大工具调用轮数 %d", maxRounds)
}

// EnsureInit 若当前会话尚未初始化，则调用房源重置（新 Session 时由调用方在首轮前调用一次）
func (a *Agent) EnsureInit() error {
	_, err := ExecuteTool(a.client, "houses_init", "{}")
	return err
}

// RunWithSessionAndToolResults 按大赛规范：使用 model_ip 与 session_id 调用模型，并收集 tool_results。
// modelIP 为模型资源 IP，端口固定 8888；sessionID 作为 Session-ID 请求头传给模型接口。
// 返回的 response 在房源查询场景下会被规范化为 JSON 字符串 {"message":"...","houses":[...]}。
func (a *Agent) RunWithSessionAndToolResults(messages []ChatMessage, userInput string, modelIP string, sessionID string) (reply string, newMessages []ChatMessage, toolResults []ToolResult, err error) {
	msgs := make([]ChatMessage, 0, len(messages)+1)
	msgs = append(msgs, messages...)
	msgs = append(msgs, ChatMessage{Role: "user", Content: userInput})
	hasSystem := false
	for _, m := range msgs {
		if m.Role == "system" {
			hasSystem = true
			break
		}
	}
	if !hasSystem {
		inner := make([]ChatMessage, 0, len(msgs)+1)
		inner = append(inner, ChatMessage{Role: "system", Content: systemPrompt})
		inner = append(inner, msgs...)
		msgs = inner
	}
	baseURL := "http://" + modelIP + ":8888"
	toolResults = make([]ToolResult, 0)

	maxRounds := 15
	for round := 0; round < maxRounds; round++ {
		msg, chatErr := a.llm.ChatWithSession(baseURL, sessionID, msgs, a.tools)
		if chatErr != nil {
			return "", nil, toolResults, chatErr
		}
		msgs = append(msgs, msg)
		if len(msg.ToolCalls) == 0 {
			reply = msg.Content
			reply = NormalizeHouseResponse(reply)
			return reply, msgs, toolResults, nil
		}
		for _, tc := range msg.ToolCalls {
			name := tc.Function.Name
			result, execErr := ExecuteTool(a.client, tc.Function.Name, tc.Function.Arguments)
			success := execErr == nil
			if execErr != nil {
				result = "错误: " + execErr.Error()
			}
			toolResults = append(toolResults, ToolResult{Name: name, Success: success, Output: result})
			msgs = append(msgs, ChatMessage{
				Role:       "tool",
				ToolCallID: tc.ID,
				Content:    result,
			})
		}
	}
	return "", nil, toolResults, fmt.Errorf("超过最大工具调用轮数 %d", maxRounds)
}

// houseIDRe 匹配房源 ID，如 HF_2001
var houseIDRe = regexp.MustCompile(`HF_[A-Za-z0-9_]+`)

// NormalizeHouseResponse 确保房源查询完成时 response 为规范 JSON：{"message":"...","houses":["HF_xxx",...]}
func NormalizeHouseResponse(reply string) string {
	reply = trimSpace(reply)
	// 若已是合法 JSON 且含 houses 字段，直接返回
	var m struct {
		Message string   `json:"message"`
		Houses  []string `json:"houses"`
	}
	if err := json.Unmarshal([]byte(reply), &m); err == nil {
		return reply
	}
	// 尝试从回复中提取 HF_xxx 并组装规范 JSON
	ids := houseIDRe.FindAllString(reply, 5)
	if len(ids) == 0 {
		return reply
	}
	out := map[string]interface{}{"message": "为您找到以下符合条件的房源。", "houses": ids}
	b, _ := json.Marshal(out)
	return string(b)
}

func trimSpace(s string) string {
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n') {
		start++
	}
	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n') {
		end--
	}
	return s[start:end]
}

// ChatMessage 对外暴露，便于 main 保存历史
type ChatMessageExport struct {
	Role    string `json:"role"`
	Content string `json:"content,omitempty"`
}

// MessagesToExport 将内部消息转为可序列化结构（仅保留 user/assistant 的 content，便于存储）
func MessagesToExport(msgs []ChatMessage) []ChatMessageExport {
	var out []ChatMessageExport
	for _, m := range msgs {
		if m.Role != "user" && m.Role != "assistant" {
			continue
		}
		out = append(out, ChatMessageExport{Role: m.Role, Content: m.Content})
	}
	return out
}

// ExportToMessages 从导出结构恢复为带 system 的完整历史（仅用于延续对话，不含 tool 消息）
func ExportToMessages(exported []ChatMessageExport, systemContent string) []ChatMessage {
	var msgs []ChatMessage
	if systemContent != "" {
		msgs = append(msgs, ChatMessage{Role: "system", Content: systemContent})
	}
	for _, e := range exported {
		msgs = append(msgs, ChatMessage{Role: e.Role, Content: e.Content})
	}
	return msgs
}
