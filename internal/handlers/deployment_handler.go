package handlers

import (
	"io"
	"net/http"

	"bloope/internal/models"
	"bloope/internal/services"

	"github.com/gin-gonic/gin"
)

type DeploymentHandler struct {
	service *services.DeploymentService
}

type createDeploymentRequest struct {
	RepoURL string            `json:"repo_url" binding:"required"`
	EnvVars map[string]string `json:"env_vars"`
}

func NewDeploymentHandler(service *services.DeploymentService) *DeploymentHandler {
	return &DeploymentHandler{
		service: service,
	}
}

func (h *DeploymentHandler) RegisterRoutes(router gin.IRouter) {
	router.POST("/deployments", h.CreateDeployment)
	router.GET("/deployments", h.ListDeployments)
	router.GET("/deployments/:id", h.GetDeployment)
	router.GET("/deployments/:id/logs", h.GetDeploymentLogs)
	router.GET("/deployments/:id/logs/stream", h.StreamDeploymentLogs)
}

func (h *DeploymentHandler) CreateDeployment(c *gin.Context) {
	var request createDeploymentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "repo_url is required"})
		return
	}

	deployment, err := h.service.CreateDeployment(request.RepoURL, request.EnvVars)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, deployment)
}

func (h *DeploymentHandler) ListDeployments(c *gin.Context) {
	deployments, err := h.service.ListDeployments()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, deployments)
}

func (h *DeploymentHandler) GetDeployment(c *gin.Context) {
	deployment, ok, err := h.service.GetDeployment(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": services.ErrDeploymentNotFound.Error()})
		return
	}

	c.JSON(http.StatusOK, deployment)
}

func (h *DeploymentHandler) GetDeploymentLogs(c *gin.Context) {
	logs, ok, err := h.service.GetDeploymentLogs(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": services.ErrDeploymentNotFound.Error()})
		return
	}

	c.JSON(http.StatusOK, logs)
}

func (h *DeploymentHandler) StreamDeploymentLogs(c *gin.Context) {
	deploymentID := c.Param("id")
	liveLogs, unsubscribe, ok := h.service.SubscribeDeploymentLogs(deploymentID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": services.ErrDeploymentNotFound.Error()})
		return
	}
	defer unsubscribe()

	historicalLogs, _, _ := h.service.GetDeploymentLogs(deploymentID)

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	for _, logEntry := range historicalLogs {
		writeDeploymentLogEvent(c, logEntry)
	}
	c.Writer.Flush()

	c.Stream(func(w io.Writer) bool {
		select {
		case <-c.Request.Context().Done():
			return false
		case logEntry, ok := <-liveLogs:
			if !ok {
				return false
			}

			writeDeploymentLogEvent(c, logEntry)
			return true
		}
	})
}

func writeDeploymentLogEvent(c *gin.Context, logEntry *models.DeploymentLog) {
	c.SSEvent("log", logEntry)
}
