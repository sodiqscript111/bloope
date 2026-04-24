package state

import (
	"fmt"

	"bloope/internal/models"
)

type Engine struct {
	allowedTransitions map[models.DeploymentStatus]map[models.DeploymentStatus]bool
}

func NewEngine() *Engine {
	return &Engine{
		allowedTransitions: map[models.DeploymentStatus]map[models.DeploymentStatus]bool{
			models.StatusPending: {
				models.StatusBuilding: true,
				models.StatusFailed:   true,
			},
			models.StatusBuilding: {
				models.StatusDeploying: true,
				models.StatusFailed:    true,
			},
			models.StatusDeploying: {
				models.StatusRunning: true,
				models.StatusFailed:  true,
			},
		},
	}
}

func (e *Engine) CanTransition(current, next models.DeploymentStatus) bool {
	nextStatuses, ok := e.allowedTransitions[current]
	if !ok {
		return false
	}

	return nextStatuses[next]
}

func (e *Engine) ValidateTransition(current, next models.DeploymentStatus) error {
	if e.CanTransition(current, next) {
		return nil
	}

	return fmt.Errorf("invalid status transition from %q to %q", current, next)
}
