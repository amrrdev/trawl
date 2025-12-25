package handler

import (
	"net/http"

	"github.com/amrrdev/trawl/services/search/internal/service"
	"github.com/gin-gonic/gin"
)

type SearchHandler struct {
	searchService *service.Search
}

func NewSearchHandler(searchService *service.Search) *SearchHandler {
	return &SearchHandler{
		searchService: searchService,
	}
}

type SearchRequest struct {
	Query string `json:"query" binding:"required"`
}

type SearchResponse struct {
	Results []service.SearchResult `json:"results"`
}

func (h *SearchHandler) Search(c *gin.Context) {
	var req SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	results, err := h.searchService.Search(c.Request.Context(), req.Query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, SearchResponse{Results: results})
}
