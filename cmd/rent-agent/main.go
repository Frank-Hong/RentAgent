// 智能租房 Agent：按设计目标与《Agent 对外接口规范和模型调用接口说明》实现。
// 提供 POST /api/v1/chat，模型通过 model_ip:8888/v1/chat/completions 调用，Session-ID 为 session_id。
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"rentagent/internal/agent"
	"rentagent/internal/api"
	"rentagent/internal/config"
)

func main() {
	cfg := config.LoadFromEnv()

	// 查询房源信息
	client := api.NewClient(cfg.RentAPIBaseURL, cfg.XUserID)
	llm := agent.NewLLMClient(cfg.LLMBaseURL, cfg.LLMAPIKey, cfg.LLMModel)
	ag := agent.New(llm, client)

	sessions := &sessionStore{byID: make(map[string][]agent.ChatMessage)}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/chat", func(w http.ResponseWriter, r *http.Request) {
		handleChat(w, r, ag, sessions)
	})
	addr := fmt.Sprintf(":%d", cfg.Port)
	server := &http.Server{Addr: addr, Handler: mux}
	if err := server.ListenAndServe(); err != nil {
		panic(err)
	}
}

type sessionStore struct {
	mu   sync.Mutex
	byID map[string][]agent.ChatMessage
}

func (s *sessionStore) get(sessionID string) []agent.ChatMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.byID[sessionID]
}

func (s *sessionStore) set(sessionID string, msgs []agent.ChatMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byID[sessionID] = msgs
}

func (s *sessionStore) isNew(sessionID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.byID[sessionID]
	return !ok
}

type chatRequest struct {
	ModelIP   string `json:"model_ip"`
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

type chatResponse struct {
	SessionID   string             `json:"session_id"`
	Response    string             `json:"response"`
	Status      string             `json:"status"`
	ToolResults []agent.ToolResult `json:"tool_results"`
	Timestamp   int64              `json:"timestamp"`
	DurationMs  int64              `json:"duration_ms"`
}

func handleChat(w http.ResponseWriter, r *http.Request, ag *agent.Agent, sessions *sessionStore) {
	start := time.Now()
	w.Header().Set("Content-Type", "application/json")
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(chatResponse{SessionID: req.SessionID, Status: "error", Response: "无效请求体", Timestamp: time.Now().Unix(), DurationMs: time.Since(start).Milliseconds()})
		return
	}
	if req.ModelIP == "" || req.SessionID == "" || req.Message == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(chatResponse{SessionID: req.SessionID, Status: "error", Response: "缺少 model_ip / session_id / message", Timestamp: time.Now().Unix(), DurationMs: time.Since(start).Milliseconds()})
		return
	}
	// 新 session：房源数据重置
	if sessions.isNew(req.SessionID) {
		_ = ag.EnsureInit()
	}
	messages := sessions.get(req.SessionID)
	reply, newMsgs, toolResults, err := ag.RunWithSessionAndToolResults(messages, req.Message, req.ModelIP, req.SessionID)
	durationMs := time.Since(start).Milliseconds()
	timestamp := time.Now().Unix()
	if err != nil {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(chatResponse{
			SessionID:   req.SessionID,
			Response:    "处理异常: " + err.Error(),
			Status:      "error",
			ToolResults: toolResults,
			Timestamp:   timestamp,
			DurationMs:  durationMs,
		})
		return
	}
	sessions.set(req.SessionID, newMsgs)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(chatResponse{
		SessionID:   req.SessionID,
		Response:    reply,
		Status:      "success",
		ToolResults: toolResults,
		Timestamp:   timestamp,
		DurationMs:  durationMs,
	})
}
