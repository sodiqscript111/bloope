package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bloope/internal/handlers"
	"bloope/internal/services"

	"github.com/gin-gonic/gin"
)

func main() {
	store, err := services.NewDeploymentStore()
	if err != nil {
		log.Fatalf("store failed: %v", err)
	}
	defer store.Close()

	service := services.NewDeploymentService(store)
	syncCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := service.SyncCaddyRoutes(syncCtx); err != nil {
		log.Printf("could not sync caddy routes on startup: %v", err)
	}

	deploymentHandler := handlers.NewDeploymentHandler(service)

	router := gin.Default()
	deploymentHandler.RegisterRoutes(router)
	deploymentHandler.RegisterRoutes(router.Group("/api"))
	registerFrontend(router)

	log.Println("Starting Bloope deployment API on :8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func registerFrontend(router *gin.Engine) {
	frontendDir := strings.TrimSpace(os.Getenv("BLOOPE_FRONTEND_DIR"))
	if frontendDir == "" {
		frontendDir = filepath.Join("frontend", "dist")
	}

	indexPath := filepath.Join(frontendDir, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		log.Printf("frontend static files not found at %s; serving API only", frontendDir)
		return
	}

	router.Static("/assets", filepath.Join(frontendDir, "assets"))
	router.GET("/", func(c *gin.Context) {
		c.File(indexPath)
	})
	router.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		c.File(indexPath)
	})
}
