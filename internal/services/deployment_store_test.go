package services

import (
	"path/filepath"
	"testing"
	"time"

	"bloope/internal/models"
)

func TestDeploymentStorePersistsDeploymentsAndLogs(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "bloope.db")
	store, err := NewDeploymentStoreAt(dbPath)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	now := time.Now().UTC()
	created, err := store.Create(&models.Deployment{
		ID:                  "dep_test",
		RepoURL:             "https://github.com/owner/repo",
		Status:              models.StatusPending,
		DetectedProjectType: "Go",
		EnvVars: map[string]string{
			"DATABASE_URL": "postgres://example",
			"REDIS_URL":    "redis://example",
		},
		ReadinessHints: []string{"Detected Go project via go.mod"},
		CreatedAt:      now,
		UpdatedAt:      now,
	})
	if err != nil {
		t.Fatalf("create deployment: %v", err)
	}
	if created.ID != "dep_test" {
		t.Fatalf("expected created deployment id to round-trip")
	}

	if err := store.AddLog(&models.DeploymentLog{
		DeploymentID: "dep_test",
		Message:      "hello",
		Timestamp:    now,
	}); err != nil {
		t.Fatalf("add log: %v", err)
	}

	if err := store.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	reopened, err := NewDeploymentStoreAt(dbPath)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer reopened.Close()

	deployment, ok, err := reopened.GetByID("dep_test")
	if err != nil {
		t.Fatalf("get deployment: %v", err)
	}
	if !ok {
		t.Fatal("expected deployment after reopening sqlite")
	}
	if len(deployment.ReadinessHints) != 1 || deployment.ReadinessHints[0] != "Detected Go project via go.mod" {
		t.Fatalf("expected readiness hints to persist, got %#v", deployment.ReadinessHints)
	}
	if len(deployment.EnvVarKeys) != 2 || deployment.EnvVarKeys[0] != "DATABASE_URL" || deployment.EnvVarKeys[1] != "REDIS_URL" {
		t.Fatalf("expected env var keys to persist without values, got %#v", deployment.EnvVarKeys)
	}

	envVars, err := reopened.GetEnvVars("dep_test")
	if err != nil {
		t.Fatalf("get env vars: %v", err)
	}
	if envVars["DATABASE_URL"] != "postgres://example" || envVars["REDIS_URL"] != "redis://example" {
		t.Fatalf("expected env var values to persist for runtime injection, got %#v", envVars)
	}

	logs, ok, err := reopened.GetLogsByDeploymentID("dep_test")
	if err != nil {
		t.Fatalf("get logs: %v", err)
	}
	if !ok || len(logs) != 1 || logs[0].Message != "hello" {
		t.Fatalf("expected persisted log, got ok=%v logs=%#v", ok, logs)
	}
}
