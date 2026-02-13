package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"myblog_last_new/internal/response"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultDeepSeekBaseURL = "https://api.deepseek.com"
	defaultDeepSeekModel   = "deepseek-chat"
)

// AIHandler handles AI proxy requests.
type AIHandler struct {
	apiKey       string
	baseURL      string
	defaultModel string
	httpClient   *http.Client
}

// ChatMessage represents a single chat message.
type ChatMessage struct {
	Role    string `json:"role" example:"user"`
	Content string `json:"content" example:"请帮我总结这篇文章"`
}

// ChatRequest represents user input for DeepSeek chat completion.
type ChatRequest struct {
	Message     string        `json:"message,omitempty" example:"帮我生成一个博客标题"`
	System      string        `json:"system,omitempty" example:"你是一个技术博客写作助手"`
	Messages    []ChatMessage `json:"messages,omitempty"`
	Model       string        `json:"model,omitempty" example:"deepseek-chat"`
	Temperature *float64      `json:"temperature,omitempty" example:"0.7"`
	MaxTokens   *int          `json:"max_tokens,omitempty" example:"512"`
}

type deepSeekErrorResponse struct {
	Error struct {
		Message string      `json:"message"`
		Type    string      `json:"type"`
		Code    interface{} `json:"code"`
	} `json:"error"`
}

type deepSeekChatResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int         `json:"index"`
		Message      ChatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// NewAIHandler creates a new AI handler.
func NewAIHandler() *AIHandler {
	return &AIHandler{
		apiKey:       strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY")),
		baseURL:      strings.TrimRight(getEnvOrDefault("DEEPSEEK_BASE_URL", defaultDeepSeekBaseURL), "/"),
		defaultModel: strings.TrimSpace(getEnvOrDefault("DEEPSEEK_MODEL", defaultDeepSeekModel)),
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Chat godoc
// @Summary DeepSeek 对话代理
// @Description 服务端代理 DeepSeek Chat Completions，避免前端暴露 API Key
// @Tags ai
// @Accept json
// @Produce json
// @Param body body ChatRequest true "对话请求"
// @Success 200 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse "参数错误"
// @Failure 500 {object} response.APIResponse "服务错误"
// @Router /ai/chat [post]
func (h *AIHandler) Chat(w http.ResponseWriter, r *http.Request) {
	if h.apiKey == "" {
		response.InternalError(w, "DeepSeek API is not configured. Please set DEEPSEEK_API_KEY.")
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	messages, err := buildChatMessages(req)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = h.defaultModel
	}

	payload := map[string]interface{}{
		"model":    model,
		"messages": messages,
		"stream":   false,
	}

	if req.Temperature != nil {
		payload["temperature"] = *req.Temperature
	}
	if req.MaxTokens != nil {
		payload["max_tokens"] = *req.MaxTokens
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		response.InternalError(w, "Failed to build request payload")
		return
	}

	httpReq, err := http.NewRequest(http.MethodPost, h.baseURL+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		response.InternalError(w, "Failed to create DeepSeek request")
		return
	}
	httpReq.Header.Set("Authorization", "Bearer "+h.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := h.httpClient.Do(httpReq)
	if err != nil {
		response.InternalError(w, "Failed to call DeepSeek API: "+err.Error())
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		response.InternalError(w, "Failed to read DeepSeek response")
		return
	}

	if resp.StatusCode != http.StatusOK {
		msg := "DeepSeek API request failed"
		var errResp deepSeekErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err == nil && strings.TrimSpace(errResp.Error.Message) != "" {
			msg = "DeepSeek API error: " + errResp.Error.Message
		}
		response.Error(w, resp.StatusCode, resp.StatusCode, msg)
		return
	}

	var chatResp deepSeekChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		response.InternalError(w, "Failed to parse DeepSeek response")
		return
	}
	if len(chatResp.Choices) == 0 {
		response.InternalError(w, "DeepSeek returned empty choices")
		return
	}

	response.Success(w, map[string]interface{}{
		"id":            chatResp.ID,
		"model":         chatResp.Model,
		"message":       chatResp.Choices[0].Message,
		"content":       chatResp.Choices[0].Message.Content,
		"finish_reason": chatResp.Choices[0].FinishReason,
		"usage":         chatResp.Usage,
	})
}

func buildChatMessages(req ChatRequest) ([]ChatMessage, error) {
	messages := make([]ChatMessage, 0, len(req.Messages)+2)

	systemPrompt := strings.TrimSpace(req.System)
	if systemPrompt != "" {
		messages = append(messages, ChatMessage{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	for _, m := range req.Messages {
		role := strings.TrimSpace(m.Role)
		content := strings.TrimSpace(m.Content)
		if role == "" || content == "" {
			return nil, errBadRequest("messages[*].role and messages[*].content are required")
		}
		if !isSupportedRole(role) {
			return nil, errBadRequest("messages[*].role must be one of: system, user, assistant")
		}
		messages = append(messages, ChatMessage{
			Role:    role,
			Content: content,
		})
	}

	if userMessage := strings.TrimSpace(req.Message); userMessage != "" {
		messages = append(messages, ChatMessage{
			Role:    "user",
			Content: userMessage,
		})
	}

	if len(messages) == 0 {
		return nil, errBadRequest(`message or messages is required, e.g. {"message":"帮我生成3个Go博客标题"}`)
	}

	return messages, nil
}

func isSupportedRole(role string) bool {
	switch role {
	case "system", "user", "assistant":
		return true
	default:
		return false
	}
}

type badRequestError struct {
	message string
}

func (e badRequestError) Error() string {
	return e.message
}

func errBadRequest(message string) error {
	return badRequestError{message: message}
}
