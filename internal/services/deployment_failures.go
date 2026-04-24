package services

import "fmt"

func (s *DeploymentService) logPipelineError(id string, err error, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	s.logDeployment(id, "%s: %v", message, err)
}

func (s *DeploymentService) failDeploymentStep(id string, err error, format string, args ...any) {
	s.logPipelineError(id, err, format, args...)
	s.markDeploymentFailed(id, err)
}

func (s *DeploymentService) failDeploymentStepWithCleanup(id string, cleanup func(), err error, format string, args ...any) {
	if cleanup != nil {
		cleanup()
	}

	s.failDeploymentStep(id, err, format, args...)
}

func (s *DeploymentService) failDeploymentStepWithCleanupAndResync(id string, cleanup func(), err error, format string, args ...any) {
	s.failDeploymentStepWithCleanup(id, cleanup, err, format, args...)
	s.syncCaddyRoutesBestEffort(id)
}

func (s *DeploymentService) markDeploymentFailed(id string, err error) {
	if err == nil {
		return
	}

	if _, failErr := s.FailDeployment(id, err.Error()); failErr != nil {
		s.logDeployment(id, "could not mark deployment failed: %v", failErr)
	}
}
