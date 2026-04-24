package services

import (
	"errors"
	"fmt"
	"log"
	"time"

	"bloope/internal/models"
)

func (s *DeploymentService) deploymentLogger(id string) func(string, ...any) {
	return func(format string, args ...any) {
		s.logDeployment(id, format, args...)
	}
}

func (s *DeploymentService) logDeployment(id string, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	log.Printf("[deployment:%s] %s", id, message)

	logEntry := &models.DeploymentLog{
		DeploymentID: id,
		Message:      message,
		Timestamp:    time.Now().UTC(),
	}

	if err := s.store.AddLog(logEntry); err != nil {
		if !errors.Is(err, ErrDeploymentNotFound) {
			log.Printf("[deployment:%s] could not persist log: %v", id, err)
		}
	}
	s.logBroker.Broadcast(logEntry)
}
