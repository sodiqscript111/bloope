package handlers

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"bloope/internal/services"

	"github.com/gin-gonic/gin"
)

func TestGetDeploymentLogsRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store, err := services.NewDeploymentStoreAt(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	defer store.Close()
	service := services.NewDeploymentService(store)
	handler := NewDeploymentHandler(service)

	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/deployments/missing/logs", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, response.Code)
	}
}

func TestStreamDeploymentLogsRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store, err := services.NewDeploymentStoreAt(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	defer store.Close()
	service := services.NewDeploymentService(store)
	handler := NewDeploymentHandler(service)

	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/deployments/missing/logs/stream", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, response.Code)
	}
}
