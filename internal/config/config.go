package config

import (
	"os"
	"strconv"
)

// Config 智能租房 Agent 配置
type Config struct {
	// 租房仿真 API
	RentAPIBaseURL string // 如 http://7.221.6.201:8080
	XUserID        string // 用户工号，请求头 X-User-ID，必须与比赛平台注册一致

	// Agent 对外服务（规范：Base URL http://localhost:8191）
	Port int // 监听端口，默认 8191

	// LLM 由大赛通过 model_ip 下发，端口 8888；本地调测可用
	LLMAPIKey  string
	LLMBaseURL string
	LLMModel   string
}

// LoadFromEnv 从环境变量加载配置
func LoadFromEnv() *Config {
	c := &Config{
		RentAPIBaseURL: "http://7.221.6.201:8080",
		Port:           8191,
		LLMBaseURL:     "https://api.openai.com/v1",
		LLMModel:       "gpt-4o-mini",
	}
	if v := os.Getenv("RENT_API_BASE_URL"); v != "" {
		c.RentAPIBaseURL = v
	}
	if v := os.Getenv("X_USER_ID"); v != "" {
		c.XUserID = v
	}
	if v := os.Getenv("AGENT_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			c.Port = p
		}
	}
	if v := os.Getenv("LLM_API_KEY"); v != "" {
		c.LLMAPIKey = v
	}
	if v := os.Getenv("LLM_BASE_URL"); v != "" {
		c.LLMBaseURL = v
	}
	if v := os.Getenv("LLM_MODEL"); v != "" {
		c.LLMModel = v
	}
	return c
}
