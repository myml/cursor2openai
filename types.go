package main

import (
	"encoding/json"
	"fmt"
)

// OpenAI API 请求和响应结构体

// ChatCompletionRequest 聊天完成请求
type ChatCompletionRequest struct {
	Model            string      `json:"model"`
	Messages         []Message   `json:"messages"`
	MaxTokens        *int        `json:"max_tokens,omitempty"`
	Temperature      *float64    `json:"temperature,omitempty"`
	TopP             *float64    `json:"top_p,omitempty"`
	N                *int        `json:"n,omitempty"`
	Stream           *bool       `json:"stream,omitempty"`
	Stop             interface{} `json:"stop,omitempty"`
	PresencePenalty  *float64    `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float64    `json:"frequency_penalty,omitempty"`
	User             string      `json:"user,omitempty"`
}

// Message 消息结构
type Message struct {
	Role    string           `json:"role"`
	Content []MessageContent `json:"content"`
}

func (f *Message) UnmarshalJSON(data []byte) error {
	var v struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(data, &v); err == nil {
		f.Role = v.Role
		f.Content = []MessageContent{{Type: "text", Text: v.Content}}
		return nil
	}
	var v2 struct {
		Role    string           `json:"role"`
		Content []MessageContent `json:"content"`
	}
	if err := json.Unmarshal(data, &v2); err == nil {
		f.Role = v2.Role
		f.Content = v2.Content
		return nil
	}
	return fmt.Errorf("invalid message content: %s", string(data))
}

type MessageContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ChatCompletionResponse 聊天完成响应
type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice 选择结构
type Choice struct {
	Index        int           `json:"index"`
	Message      ChoiceMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type ChoiceMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Usage 使用情况
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// EmbeddingRequest 嵌入请求
type EmbeddingRequest struct {
	Input string `json:"input"`
	Model string `json:"model"`
	User  string `json:"user,omitempty"`
}

// EmbeddingResponse 嵌入响应
type EmbeddingResponse struct {
	Object string      `json:"object"`
	Data   []Embedding `json:"data"`
	Model  string      `json:"model"`
	Usage  Usage       `json:"usage"`
}

// Embedding 嵌入数据
type Embedding struct {
	Object    string    `json:"object"`
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index"`
}

// ModelsResponse 模型列表响应
type ModelsResponse struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

// Model 模型信息
type Model struct {
	ID         string       `json:"id"`
	Object     string       `json:"object"`
	Created    int64        `json:"created"`
	OwnedBy    string       `json:"owned_by"`
	Permission []Permission `json:"permission"`
	Root       string       `json:"root"`
	Parent     string       `json:"parent"`
}

// Permission 权限信息
type Permission struct {
	ID                 string `json:"id"`
	Object             string `json:"object"`
	Created            int64  `json:"created"`
	AllowCreateEngine  bool   `json:"allow_create_engine"`
	AllowSampling      bool   `json:"allow_sampling"`
	AllowLogprobs      bool   `json:"allow_logprobs"`
	AllowSearchIndices bool   `json:"allow_search_indices"`
	AllowView          bool   `json:"allow_view"`
	AllowFineTuning    bool   `json:"allow_fine_tuning"`
	Organization       string `json:"organization"`
	Group              string `json:"group"`
	IsBlocking         bool   `json:"is_blocking"`
}

// ChatCompletionStreamResponse 流式聊天完成响应
type ChatCompletionStreamResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []StreamChoice `json:"choices"`
}

// StreamChoice 流式选择结构
type StreamChoice struct {
	Index        int         `json:"index"`
	Delta        StreamDelta `json:"delta"`
	FinishReason *string     `json:"finish_reason,omitempty"`
}

// StreamDelta 流式增量结构
type StreamDelta struct {
	Role    *string `json:"role,omitempty"`
	Content *string `json:"content,omitempty"`
}
