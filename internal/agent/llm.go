package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// ChatMessage OpenAI 对话消息
type ChatMessage struct {
	Role         string         `json:"role"`
	Content      string         `json:"content,omitempty"`
	ToolCalls    []ToolCall     `json:"tool_calls,omitempty"`
	ToolCallID   string         `json:"tool_call_id,omitempty"`
	Name         string         `json:"name,omitempty"`
}

// ToolCall 模型返回的工具调用
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// ChatRequest 请求体
type ChatRequest struct {
	Model       string       `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Tools       []ToolDef    `json:"tools,omitempty"`
	ToolChoice  string       `json:"tool_choice,omitempty"` // "auto" 或 "none"
}

// ChatResponse 响应体
type ChatResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Index   int         `json:"index"`
		Message ChatMessage `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// LLMClient OpenAI 兼容的聊天接口
type LLMClient struct {
	BaseURL string
	APIKey  string
	Model   string
	HTTP    *http.Client
}

// NewLLMClient 创建 LLM 客户端
func NewLLMClient(baseURL, apiKey, model string) *LLMClient {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &LLMClient{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   model,
		HTTP:    &http.Client{},
	}
}

// Chat 发送对话请求，返回助手消息（可能含 tool_calls）
// 使用 client 的 BaseURL、APIKey；若需按规范使用 model_ip + Session-ID，请用 ChatWithSession。
func (c *LLMClient) Chat(messages []ChatMessage, tools []ToolDef) (ChatMessage, error) {
	return c.ChatWithSession(c.BaseURL, "", messages, tools)
}

// ChatWithSession 按大赛规范：baseURL 为 http://{model_ip}:8888，请求头带 Session-ID（评测会话 ID）
func (c *LLMClient) ChatWithSession(baseURL, sessionID string, messages []ChatMessage, tools []ToolDef) (ChatMessage, error) {
	if baseURL == "" {
		baseURL = c.BaseURL
	}
	reqBody := ChatRequest{
		Model:      c.Model,
		Messages:   messages,
		Tools:      tools,
		ToolChoice: "auto",
	}
	if len(tools) == 0 {
		reqBody.ToolChoice = ""
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return ChatMessage{}, err
	}
	// 规范：模型接口路径为 /v1/chat/completions
	url := strings.TrimSuffix(baseURL, "/") + "/v1/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return ChatMessage{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if sessionID != "" {
		req.Header.Set("Session-ID", sessionID)
	}
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return ChatMessage{}, err
	}
	defer resp.Body.Close()
	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return ChatMessage{}, fmt.Errorf("解码响应失败: %w", err)
	}
	if chatResp.Error != nil {
		return ChatMessage{}, fmt.Errorf("API 错误: %s", chatResp.Error.Message)
	}
	if len(chatResp.Choices) == 0 {
		return ChatMessage{}, fmt.Errorf("无返回内容")
	}
	return chatResp.Choices[0].Message, nil
}
