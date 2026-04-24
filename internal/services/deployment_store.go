package services

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"bloope/internal/models"

	_ "modernc.org/sqlite"
)

var ErrDeploymentNotFound = errors.New("deployment not found")

type DeploymentStore struct {
	db *sql.DB
}

func NewDeploymentStore() (*DeploymentStore, error) {
	dbPath := os.Getenv("BLOOPE_DB_PATH")
	if dbPath == "" {
		dbPath = filepath.Join("data", "bloope.db")
	}

	return NewDeploymentStoreAt(dbPath)
}

func NewDeploymentStoreAt(dbPath string) (*DeploymentStore, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}
	db.SetMaxOpenConns(1)

	store := &DeploymentStore{db: db}
	if err := store.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *DeploymentStore) Close() error {
	return s.db.Close()
}

func (s *DeploymentStore) migrate() error {
	statements := []string{
		`PRAGMA journal_mode = WAL;`,
		`PRAGMA busy_timeout = 5000;`,
		`PRAGMA foreign_keys = ON;`,
		`CREATE TABLE IF NOT EXISTS deployments (
			id TEXT PRIMARY KEY,
			repo_url TEXT NOT NULL,
			status TEXT NOT NULL,
			image_tag TEXT NOT NULL DEFAULT '',
			live_url TEXT NOT NULL DEFAULT '',
			error_message TEXT NOT NULL DEFAULT '',
			detected_project_type TEXT NOT NULL DEFAULT '',
			detected_framework TEXT NOT NULL DEFAULT '',
			start_command TEXT NOT NULL DEFAULT '',
			readiness_hints TEXT NOT NULL DEFAULT '[]',
			container_name TEXT NOT NULL DEFAULT '',
			container_id TEXT NOT NULL DEFAULT '',
			host_port INTEGER NOT NULL DEFAULT 0,
			source_path TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS deployment_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			deployment_id TEXT NOT NULL,
			message TEXT NOT NULL,
			timestamp TEXT NOT NULL,
			FOREIGN KEY (deployment_id) REFERENCES deployments(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS deployment_env_vars (
			deployment_id TEXT NOT NULL,
			name TEXT NOT NULL,
			value TEXT NOT NULL,
			PRIMARY KEY (deployment_id, name),
			FOREIGN KEY (deployment_id) REFERENCES deployments(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_deployment_logs_deployment_id_id ON deployment_logs(deployment_id, id);`,
		`CREATE INDEX IF NOT EXISTS idx_deployment_env_vars_deployment_id ON deployment_env_vars(deployment_id);`,
		`CREATE INDEX IF NOT EXISTS idx_deployments_created_at ON deployments(created_at);`,
		`ALTER TABLE deployments ADD COLUMN detected_framework TEXT NOT NULL DEFAULT '';`,
		`ALTER TABLE deployments ADD COLUMN start_command TEXT NOT NULL DEFAULT '';`,
	}

	for _, statement := range statements {
		if _, err := s.db.Exec(statement); err != nil {
			if strings.Contains(err.Error(), "duplicate column name") {
				continue
			}
			return fmt.Errorf("run sqlite migration: %w", err)
		}
	}

	return nil
}

func (s *DeploymentStore) Create(deployment *models.Deployment) (*models.Deployment, error) {
	hintsJSON, err := json.Marshal(deployment.ReadinessHints)
	if err != nil {
		return nil, fmt.Errorf("marshal readiness hints: %w", err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin create deployment transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`INSERT INTO deployments (
			id, repo_url, status, image_tag, live_url, error_message, detected_project_type,
			detected_framework, start_command, readiness_hints, container_name, container_id, host_port,
			source_path, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		deployment.ID,
		deployment.RepoURL,
		string(deployment.Status),
		deployment.ImageTag,
		deployment.LiveURL,
		deployment.ErrorMessage,
		deployment.DetectedProjectType,
		deployment.DetectedFramework,
		deployment.StartCommand,
		string(hintsJSON),
		deployment.ContainerName,
		deployment.ContainerID,
		deployment.HostPort,
		deployment.SourcePath,
		formatTime(deployment.CreatedAt),
		formatTime(deployment.UpdatedAt),
	)
	if err != nil {
		return nil, fmt.Errorf("insert deployment: %w", err)
	}

	for _, key := range sortedEnvKeys(deployment.EnvVars) {
		if _, err := tx.Exec(
			`INSERT INTO deployment_env_vars (deployment_id, name, value) VALUES (?, ?, ?)`,
			deployment.ID,
			key,
			deployment.EnvVars[key],
		); err != nil {
			return nil, fmt.Errorf("insert deployment env var: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit create deployment transaction: %w", err)
	}

	return s.mustGetByID(deployment.ID)
}

func (s *DeploymentStore) GetByID(id string) (*models.Deployment, bool, error) {
	deployment, err := s.scanDeployment(s.db.QueryRow(`SELECT `+deploymentColumns+` FROM deployments WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if err := s.attachEnvVarKeys(deployment); err != nil {
		return nil, false, err
	}

	return deployment, true, nil
}

func (s *DeploymentStore) List() ([]*models.Deployment, error) {
	rows, err := s.db.Query(`SELECT ` + deploymentColumns + ` FROM deployments ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list deployments: %w", err)
	}

	deployments := []*models.Deployment{}
	for rows.Next() {
		deployment, err := scanDeploymentRows(rows)
		if err != nil {
			_ = rows.Close()
			return nil, err
		}
		deployments = append(deployments, deployment)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("iterate deployments: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close deployments rows: %w", err)
	}

	for _, deployment := range deployments {
		if err := s.attachEnvVarKeys(deployment); err != nil {
			return nil, err
		}
	}

	return deployments, nil
}

func (s *DeploymentStore) ListRunning() ([]*models.Deployment, error) {
	rows, err := s.db.Query(`SELECT `+deploymentColumns+` FROM deployments WHERE status = ? ORDER BY created_at ASC`, string(models.StatusRunning))
	if err != nil {
		return nil, fmt.Errorf("list running deployments: %w", err)
	}

	deployments := []*models.Deployment{}
	for rows.Next() {
		deployment, err := scanDeploymentRows(rows)
		if err != nil {
			_ = rows.Close()
			return nil, err
		}
		deployments = append(deployments, deployment)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("iterate running deployments: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close running deployments rows: %w", err)
	}

	for _, deployment := range deployments {
		if err := s.attachEnvVarKeys(deployment); err != nil {
			return nil, err
		}
	}

	return deployments, nil
}

func (s *DeploymentStore) SetStatus(id string, status models.DeploymentStatus) (*models.Deployment, error) {
	return s.updateAndGet(id, `UPDATE deployments SET status = ?, updated_at = ? WHERE id = ?`, string(status), formatTime(time.Now().UTC()), id)
}

func (s *DeploymentStore) SaveImageTag(id string, imageTag string) (*models.Deployment, error) {
	return s.updateAndGet(id, `UPDATE deployments SET image_tag = ?, updated_at = ? WHERE id = ?`, imageTag, formatTime(time.Now().UTC()), id)
}

func (s *DeploymentStore) SaveSourcePath(id string, sourcePath string) (*models.Deployment, error) {
	return s.updateAndGet(id, `UPDATE deployments SET source_path = ?, updated_at = ? WHERE id = ?`, sourcePath, formatTime(time.Now().UTC()), id)
}

func (s *DeploymentStore) SaveSourceInsights(id string, projectType string, framework string, startCommand string, hints []string) (*models.Deployment, error) {
	hintsJSON, err := json.Marshal(hints)
	if err != nil {
		return nil, fmt.Errorf("marshal readiness hints: %w", err)
	}

	return s.updateAndGet(
		id,
		`UPDATE deployments SET detected_project_type = ?, detected_framework = ?, start_command = ?, readiness_hints = ?, updated_at = ? WHERE id = ?`,
		projectType,
		framework,
		startCommand,
		string(hintsJSON),
		formatTime(time.Now().UTC()),
		id,
	)
}

func (s *DeploymentStore) SaveRuntime(id string, containerName string, containerID string, hostPort int) (*models.Deployment, error) {
	return s.updateAndGet(
		id,
		`UPDATE deployments SET container_name = ?, container_id = ?, host_port = ?, updated_at = ? WHERE id = ?`,
		containerName,
		containerID,
		hostPort,
		formatTime(time.Now().UTC()),
		id,
	)
}

func (s *DeploymentStore) Complete(id string, imageTag string, liveURL string) (*models.Deployment, error) {
	return s.updateAndGet(
		id,
		`UPDATE deployments SET status = ?, image_tag = ?, live_url = ?, updated_at = ? WHERE id = ?`,
		string(models.StatusRunning),
		imageTag,
		liveURL,
		formatTime(time.Now().UTC()),
		id,
	)
}

func (s *DeploymentStore) SaveLiveURL(id string, liveURL string) (*models.Deployment, error) {
	return s.updateAndGet(
		id,
		`UPDATE deployments SET live_url = ?, updated_at = ? WHERE id = ?`,
		liveURL,
		formatTime(time.Now().UTC()),
		id,
	)
}

func (s *DeploymentStore) Fail(id string, message string) (*models.Deployment, error) {
	return s.updateAndGet(
		id,
		`UPDATE deployments SET status = ?, error_message = ?, updated_at = ? WHERE id = ?`,
		string(models.StatusFailed),
		message,
		formatTime(time.Now().UTC()),
		id,
	)
}

func (s *DeploymentStore) AddLog(logEntry *models.DeploymentLog) error {
	result, err := s.db.Exec(
		`INSERT INTO deployment_logs (deployment_id, message, timestamp) VALUES (?, ?, ?)`,
		logEntry.DeploymentID,
		logEntry.Message,
		formatTime(logEntry.Timestamp),
	)
	if err != nil {
		return fmt.Errorf("insert deployment log: %w", err)
	}

	id, err := result.LastInsertId()
	if err == nil {
		logEntry.ID = id
	}

	return nil
}

func (s *DeploymentStore) GetEnvVars(id string) (map[string]string, error) {
	if _, ok, err := s.GetByID(id); err != nil || !ok {
		if err != nil {
			return nil, err
		}
		return nil, ErrDeploymentNotFound
	}

	rows, err := s.db.Query(`SELECT name, value FROM deployment_env_vars WHERE deployment_id = ? ORDER BY name ASC`, id)
	if err != nil {
		return nil, fmt.Errorf("list deployment env vars: %w", err)
	}
	defer rows.Close()

	envVars := map[string]string{}
	for rows.Next() {
		var key string
		var value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("scan deployment env var: %w", err)
		}
		envVars[key] = value
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate deployment env vars: %w", err)
	}

	return envVars, nil
}

func (s *DeploymentStore) GetLogsByDeploymentID(id string) ([]*models.DeploymentLog, bool, error) {
	if _, ok, err := s.GetByID(id); err != nil || !ok {
		return nil, ok, err
	}

	rows, err := s.db.Query(
		`SELECT id, deployment_id, message, timestamp FROM deployment_logs WHERE deployment_id = ? ORDER BY id ASC`,
		id,
	)
	if err != nil {
		return nil, false, fmt.Errorf("list deployment logs: %w", err)
	}
	defer rows.Close()

	logs := []*models.DeploymentLog{}
	for rows.Next() {
		var timestamp string
		logEntry := &models.DeploymentLog{}
		if err := rows.Scan(&logEntry.ID, &logEntry.DeploymentID, &logEntry.Message, &timestamp); err != nil {
			return nil, false, fmt.Errorf("scan deployment log: %w", err)
		}
		logEntry.Timestamp = parseTime(timestamp)
		logs = append(logs, logEntry)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("iterate deployment logs: %w", err)
	}

	return logs, true, nil
}

func (s *DeploymentStore) attachEnvVarKeys(deployment *models.Deployment) error {
	rows, err := s.db.Query(`SELECT name FROM deployment_env_vars WHERE deployment_id = ? ORDER BY name ASC`, deployment.ID)
	if err != nil {
		return fmt.Errorf("list deployment env var keys: %w", err)
	}
	defer rows.Close()

	deployment.EnvVarKeys = []string{}
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return fmt.Errorf("scan deployment env var key: %w", err)
		}
		deployment.EnvVarKeys = append(deployment.EnvVarKeys, key)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate deployment env var keys: %w", err)
	}

	return nil
}

func (s *DeploymentStore) updateAndGet(id string, statement string, args ...any) (*models.Deployment, error) {
	result, err := s.db.Exec(statement, args...)
	if err != nil {
		return nil, fmt.Errorf("update deployment: %w", err)
	}

	affected, err := result.RowsAffected()
	if err == nil && affected == 0 {
		return nil, ErrDeploymentNotFound
	}

	return s.mustGetByID(id)
}

func (s *DeploymentStore) mustGetByID(id string) (*models.Deployment, error) {
	deployment, ok, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrDeploymentNotFound
	}

	return deployment, nil
}

const deploymentColumns = `
	id, repo_url, status, image_tag, live_url, error_message, detected_project_type,
	detected_framework, start_command, readiness_hints, container_name, container_id,
	host_port, source_path, created_at, updated_at
`

type rowScanner interface {
	Scan(dest ...any) error
}

func (s *DeploymentStore) scanDeployment(row rowScanner) (*models.Deployment, error) {
	return scanDeployment(row)
}

func scanDeploymentRows(row rowScanner) (*models.Deployment, error) {
	return scanDeployment(row)
}

func scanDeployment(row rowScanner) (*models.Deployment, error) {
	var status string
	var hintsJSON string
	var createdAt string
	var updatedAt string
	deployment := &models.Deployment{}

	err := row.Scan(
		&deployment.ID,
		&deployment.RepoURL,
		&status,
		&deployment.ImageTag,
		&deployment.LiveURL,
		&deployment.ErrorMessage,
		&deployment.DetectedProjectType,
		&deployment.DetectedFramework,
		&deployment.StartCommand,
		&hintsJSON,
		&deployment.ContainerName,
		&deployment.ContainerID,
		&deployment.HostPort,
		&deployment.SourcePath,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return nil, err
	}

	deployment.Status = models.DeploymentStatus(status)
	if err := json.Unmarshal([]byte(hintsJSON), &deployment.ReadinessHints); err != nil {
		deployment.ReadinessHints = []string{}
	}
	deployment.CreatedAt = parseTime(createdAt)
	deployment.UpdatedAt = parseTime(updatedAt)

	return deployment, nil
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		value = time.Now().UTC()
	}

	return value.UTC().Format(time.RFC3339Nano)
}

func parseTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}

	return parsed
}

func sortedEnvKeys(envVars map[string]string) []string {
	keys := make([]string, 0, len(envVars))
	for key := range envVars {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
