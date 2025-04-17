package api

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"paste-service/internal/service"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *service.PasteService
	router  *gin.Engine
}

func NewHandler(service *service.PasteService) *Handler {
	h := &Handler{
		service: service,
	}
	h.setupRouter()
	return h
}

func (h *Handler) setupRouter() {
	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Link")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Max-Age", "300")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// TODO: rate limit

	r.GET("/", h.handleHome)
	r.GET("/health", h.handleHealth)

	api := r.Group("/api")
	{
		pastes := api.Group("/pastes")
		{
			pastes.POST("/", h.handleCreatePaste)
			pastes.GET("/top", h.handleGetTopPastes)
			pastes.GET("/recent", h.handleGetRecentPastes)
			pastes.GET("/:slug", h.handleGetPaste)
			pastes.PUT("/:slug", h.handleUpdatePaste)
		}
	}

	h.router = r
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}

func (h *Handler) handleHome(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Paste Service API",
		"version": "1.0.0",
	})
}

func (h *Handler) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

type CreatePasteRequest struct {
	Content   string         `json:"content" binding:"required"`
	Tags      []string       `json:"tags,omitempty"`
	ExpiresIn *time.Duration `json:"expires_in,omitempty"`
	AutoTag   bool           `json:"auto_tag"`
}

func (h *Handler) handleCreatePaste(c *gin.Context) {
	var req CreatePasteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Некорректный запрос"})
		return
	}

	serviceReq := service.CreatePasteRequest{
		Content:   req.Content,
		Tags:      req.Tags,
		ExpiresIn: req.ExpiresIn,
		AutoTag:   req.AutoTag,
	}

	paste, err := h.service.CreatePaste(serviceReq)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, paste)
}

func (h *Handler) handleGetPaste(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Не указан slug"})
		return
	}

	paste, err := h.service.GetPaste(slug)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, paste)
}

type UpdatePasteRequest struct {
	Content   string   `json:"content" binding:"required"`
	Tags      []string `json:"tags,omitempty"`
	EditToken string   `json:"edit_token" binding:"required"`
}

func (h *Handler) handleUpdatePaste(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Не указан slug"})
		return
	}

	var req UpdatePasteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Некорректный запрос"})
		return
	}

	paste, err := h.service.UpdatePaste(slug, req.EditToken, req.Content, req.Tags)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, paste)
}

func (h *Handler) handleGetTopPastes(c *gin.Context) {
	limit := getQueryIntParam(c, "limit", 10)

	pastes, err := h.service.GetTopPastes(limit)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, pastes)
}

func (h *Handler) handleGetRecentPastes(c *gin.Context) {
	limit := getQueryIntParam(c, "limit", 10)

	pastes, err := h.service.GetRecentPastes(limit)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, pastes)
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func handleServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidPaste):
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Некорректные данные пасты"})
	case errors.Is(err, service.ErrPasteNotFound):
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Паста не найдена"})
	case errors.Is(err, service.ErrInvalidEditToken):
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "Неверный токен редактирования"})
	case errors.Is(err, service.ErrPasteExpired):
		c.JSON(http.StatusGone, ErrorResponse{Error: "Срок действия пасты истек"})
	case errors.Is(err, service.ErrTaggerUnavailable):
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "Сервис тэггирования недоступен"})
	case errors.Is(err, service.ErrSlugGeneratorUnavailable):
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "Сервис генерации slug недоступен"})
	default:
		log.Printf("Необработанная ошибка: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Внутренняя ошибка сервера"})
	}
}

func getQueryIntParam(c *gin.Context, param string, defaultValue int) int {
	valueStr := c.Query(param)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil || value <= 0 {
		return defaultValue
	}

	return value
}
