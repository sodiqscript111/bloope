package models

import "time"

type DeploymentLog struct {
	ID           int64     `json:"id"`
	DeploymentID string    `json:"deployment_id"`
	Message      string    `json:"message"`
	Timestamp    time.Time `json:"timestamp"`
}
