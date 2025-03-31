package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ContextHandler struct {
	eApi EmulatorApi
}

func NewContextHandler(eApi EmulatorApi) *ContextHandler {
	return &ContextHandler{
		eApi: eApi,
	}
}

func (h *ContextHandler) GetAvailableContexts(c *gin.Context) {
	contexts := []string{"emulator", "ue", "gnb"}
	c.JSON(http.StatusOK, gin.H{
		"contexts": contexts,
	})
}

func (h *ContextHandler) GetContextByType(c *gin.Context) {
	contextType := c.Param("type")

	switch contextType {
	case "ue":
		ues := h.eApi.ListUes()
		c.JSON(http.StatusOK, gin.H{
			"type":    "ue",
			"objects": ues,
		})
	case "gnb":
		gnbs := h.eApi.ListGnbs()
		c.JSON(http.StatusOK, gin.H{
			"type":    "gnb",
			"objects": gnbs,
		})
	case "emulator":
		c.JSON(http.StatusOK, gin.H{
			"type":    "emulator",
			"objects": []string{"emulator"},
		})
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid context type",
		})
	}
}
