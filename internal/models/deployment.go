package models

import "time"

type DeploymentStatus string

const (
	StatusPending   DeploymentStatus = "pending"
	StatusBuilding  DeploymentStatus = "building"
	StatusDeploying DeploymentStatus = "deploying"
	StatusRunning   DeploymentStatus = "running"
	StatusFailed    DeploymentStatus = "failed"
)

type Deployment struct {
	ID                  string            `json:"id"`
	RepoURL             string            `json:"repo_url"`
	Status              DeploymentStatus  `json:"status"`
	ImageTag            string            `json:"image_tag"`
	LiveURL             string            `json:"live_url"`
	ErrorMessage        string            `json:"error_message"`
	DetectedProjectType string            `json:"detected_project_type"`
	DetectedFramework   string            `json:"detected_framework"`
	StartCommand        string            `json:"start_command"`
	EnvVars             map[string]string `json:"-"`
	EnvVarKeys          []string          `json:"env_var_keys"`
	ReadinessHints      []string          `json:"readiness_hints"`
	ContainerName       string            `json:"container_name"`
	ContainerID         string            `json:"container_id"`
	HostPort            int               `json:"host_port"`
	SourcePath          string            `json:"source_path"`
	CreatedAt           time.Time         `json:"created_at"`
	UpdatedAt           time.Time         `json:"updated_at"`
}
