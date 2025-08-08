package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func getCursorAgent() string {
	agent := os.Getenv("CURSOR_AGENT_PATH")
	if len(agent) > 0 {
		return agent
	}
	return "cursor-agent"
}

func getCursorApiKey() (string, error) {
	apiKey := os.Getenv("CURSOR_API_KEY")
	if len(apiKey) > 0 {
		return apiKey, nil
	}
	apiKeyUrl := os.Getenv("CURSOR_API_KEY_URL")
	if len(apiKeyUrl) > 0 {
		slog.Debug("get api key from url", "apiKeyUrl", apiKeyUrl)
		resp, err := http.Get(apiKeyUrl)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		return string(body), nil
	}
	apiKeyScript := os.Getenv("CURSOR_API_KEY_SCRIPT")
	if len(apiKeyScript) > 0 {
		slog.Debug("get api key from script", "apiKeyScript", apiKeyScript)
		cmd := exec.Command("bash", "-c", apiKeyScript)
		cmd.Stderr = os.Stderr
		output, err := cmd.Output()
		if err != nil {
			return "", err
		}
		return string(output), nil
	}
	return "", errors.New("CURSOR_API_KEY or CURSOR_API_KEY_URL or CURSOR_API_KEY_SCRIPT is not set")
}

// 聊天完成接口
func chatCompletionsHandler(c *gin.Context) {
	slog.Debug("chatCompletionsHandler")
	var req ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}
	slog.Debug("chatCompletionsHandler", "req", req)

	// 检查是否请求流式响应
	isStream := req.Stream != nil && *req.Stream
	if isStream {
		handleStreamChatCompletion(c, req)
	} else {
		handleNonStreamChatCompletion(c, req)
	}
}

// 处理非流式聊天完成
func handleNonStreamChatCompletion(c *gin.Context, req ChatCompletionRequest) {
	apiKey, err := getCursorApiKey()
	if err != nil {
		slog.Error("getCursorApiKey", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	cmd := exec.Command(getCursorAgent(),
		"--model", req.Model,
		"--api-key", apiKey,
		"--print",
		"--output-format", "text",
	)
	message := ""
	for _, msg := range req.Messages {
		message += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
	}
	slog.Info("chat", "input", message)
	cmd.Stdin = strings.NewReader(message)

	out, err := cmd.CombinedOutput()

	if err != nil {
		slog.Error("error", "error", err, "out", string(out))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	slog.Debug("execute cursor-agent", "input", message, "model", req.Model, "api-key", apiKey[:4]+"******", "output", string(out))
	content := string(out)
	slog.Info("chat", "output", content)
	response := ChatCompletionResponse{
		ID:      "chatcmpl-123",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: "stop",
			},
		},
		Usage: Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}

	c.JSON(http.StatusOK, response)
}

// 处理流式聊天完成
func handleStreamChatCompletion(c *gin.Context, req ChatCompletionRequest) {
	streamID := uuid.New().String()
	apiKey, err := getCursorApiKey()
	if err != nil {
		slog.Error("getCursorApiKey", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// 设置响应头为流式传输
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Cache-Control")

	r, w := io.Pipe()
	// 启动后台处理
	go func() {
		defer w.CloseWithError(io.EOF)
		cmd := exec.CommandContext(c, getCursorAgent(),
			"--model", req.Model,
			"--api-key", apiKey,
			"--print",
			"--output-format", "stream-json",
		)
		message := ""
		for _, msg := range req.Messages {
			message += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
		}
		slog.Info("chat", "input", message)
		slog.Debug("execute cursor-agent", "streamID", streamID, "input", message, "model", req.Model, "api-key", apiKey[:4]+"******")

		cmd.Stdin = strings.NewReader(message)
		cmd.Stdout = w
		cmd.Stderr = w

		if err := cmd.Start(); err != nil {
			w.CloseWithError(err)
			return
		}
		if err := cmd.Wait(); err != nil {
			w.CloseWithError(err)
			return
		}
	}()

	// 发送流式响应
	created := time.Now().Unix()
	// 发送开始事件
	startEvent := ChatCompletionStreamResponse{
		ID:      streamID,
		Object:  "chat.completion.chunk",
		Created: created,
		Model:   req.Model,
		Choices: []StreamChoice{
			{
				Index: 0,
				Delta: StreamDelta{
					Role: stringPtr("assistant"),
				},
			},
		},
	}

	if err := sendStreamEvent(c.Writer, startEvent); err != nil {
		slog.Error("Error sending start event", "error", err)
		return
	}

	reader := bufio.NewReader(r)
	output := ""
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				// 流结束，发送完成事件
				finishReason := "stop"
				endEvent := ChatCompletionStreamResponse{
					ID:      streamID,
					Object:  "chat.completion.chunk",
					Created: created,
					Model:   req.Model,
					Choices: []StreamChoice{
						{
							Index:        0,
							FinishReason: &finishReason,
						},
					},
				}
				sendStreamEvent(c.Writer, endEvent)
				slog.Info("chat", "output", output)
				return
			}
			slog.Error("Error reading from pipe", "error", err)
			return
		}
		if len(line) == 0 || line[0] != '{' {
			continue
		}
		slog.Debug("execute cursor-agent", "streamID", streamID, "output", string(line))
		// {"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"即可"}]},"session_id":"9c2e2e59-a6cf-4af2-bdbf-72c6353b6a62"}
		var streamJSON struct {
			Type    string `json:"type"`
			Message struct {
				Role    string `json:"role"`
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			} `json:"message"`
		}
		err = json.Unmarshal(line, &streamJSON)
		if err != nil {
			slog.Error("Error unmarshalling line", "error", err)
			continue
		}
		if streamJSON.Type != "assistant" {
			continue
		}
		// 发送内容块
		contentEvent := ChatCompletionStreamResponse{
			ID:      streamID,
			Object:  "chat.completion.chunk",
			Created: created,
			Model:   req.Model,
			Choices: []StreamChoice{
				{
					Index: 0,
					Delta: StreamDelta{
						Content: &streamJSON.Message.Content[0].Text,
					},
				},
			},
		}
		output += streamJSON.Message.Content[0].Text
		if err := sendStreamEvent(c.Writer, contentEvent); err != nil {
			slog.Error("Error sending content event", "error", err)
			return
		}
	}
}

// 发送流式事件
func sendStreamEvent(w gin.ResponseWriter, event ChatCompletionStreamResponse) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// 发送SSE格式的数据
	_, err = fmt.Fprintf(w, "data: %s\n\n", data)
	if err != nil {
		return err
	}

	// 刷新缓冲区
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	return nil
}

// 辅助函数：创建字符串指针
func stringPtr(s string) *string {
	return &s
}
