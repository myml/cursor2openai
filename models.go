package main

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// 模型列表接口
func modelsHandler(c *gin.Context) {
	slog.Debug("modelsHandler")
	response := ModelsResponse{
		Object: "list",
		Data:   nil,
	}
	modules := []string{"gpt-5", "sonnet-4", "sonnet-4-thinking"}
	for _, module := range modules {
		response.Data = append(response.Data, Model{
			ID:      module,
			Object:  "model",
			Created: time.Now().Unix(),
		})
	}
	c.JSON(http.StatusOK, response)
}
