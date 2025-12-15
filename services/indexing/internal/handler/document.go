package handler

import (
	"net/http"
	"strings"

	"github.com/amrrdev/trawl/services/indexing/internal/service"
	"github.com/amrrdev/trawl/services/shared/middleware"
	"github.com/gin-gonic/gin"
)

type DocumentHandler struct {
	documentService *service.Document
}

func NewDocumentHandler(documentService *service.Document) *DocumentHandler {
	return &DocumentHandler{
		documentService: documentService,
	}
}

func (h *DocumentHandler) GetUploadUrl(c *gin.Context) {
	userID := middleware.GetUserID(c)
	filename := c.Param("filename")

	if strings.TrimSpace(filename) == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "filename is required",
		})
		return
	}

	resp, err := h.documentService.GetUploadUrl(c, userID, filename)
	if err != nil {
		statusCode := http.StatusInternalServerError
		message := "Failed to generate upload URL"

		errMsg := err.Error()
		if strings.Contains(errMsg, "required") {
			statusCode = http.StatusBadRequest
			message = err.Error()
		}

		c.JSON(statusCode, gin.H{
			"error": message,
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *DocumentHandler) GetDownloadUrl(c *gin.Context) {
	userID := middleware.GetUserID(c)
	filename := c.Param("filename")

	if strings.TrimSpace(filename) == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "filename is required",
		})
		return
	}

	resp, err := h.documentService.GetDownloadUrl(c, userID, filename)
	if err != nil {
		statusCode := http.StatusInternalServerError
		message := "Failed to generate download URL"

		errMsg := err.Error()
		if strings.Contains(errMsg, "required") {
			statusCode = http.StatusBadRequest
			message = err.Error()
		} else if strings.Contains(errMsg, "not found") {
			statusCode = http.StatusNotFound
			message = "File not found"
		}

		c.JSON(statusCode, gin.H{
			"error": message,
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *DocumentHandler) ListFiles(c *gin.Context) {
	userID := middleware.GetUserID(c)

	resp, err := h.documentService.ListFiles(c, userID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		message := "Failed to list files"

		errMsg := err.Error()
		if strings.Contains(errMsg, "required") {
			statusCode = http.StatusBadRequest
			message = err.Error()
		}

		c.JSON(statusCode, gin.H{
			"error": message,
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}
