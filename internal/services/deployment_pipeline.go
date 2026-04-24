package services

import (
	"context"
	"strings"
	"time"

	buildrunner "bloope/internal/build"
	dockerruntime "bloope/internal/docker"
	"bloope/internal/models"
	"bloope/internal/source"
	"bloope/internal/validation"
)

func (s *DeploymentService) runFakeDeployment(id string) {
	logger := s.deploymentLogger(id)

	logger("queued fake deployment")
	time.Sleep(1 * time.Second)

	repoURL, ok := s.loadDeploymentForPipeline(id)
	if !ok {
		return
	}

	sourcePath, ok := s.cloneDeploymentRepository(id, repoURL)
	if !ok {
		return
	}

	inspection, ok := s.inspectDeploymentSource(id, sourcePath)
	if !ok {
		return
	}

	buildResult, ok := s.buildDeploymentImage(id, repoURL, sourcePath)
	if !ok {
		return
	}

	runtimeInfo, ok := s.startDeploymentRuntime(id, buildResult.ImageTag, inspection.StartCommand)
	if !ok {
		return
	}

	if !s.configureDeploymentIngress(id, runtimeInfo) {
		return
	}

	if !s.completeDeploymentRun(id, buildResult.ImageTag, runtimeInfo.ContainerName) {
		return
	}

	logger("deployment running at %s with image %s", s.router.LiveURL(id), buildResult.ImageTag)
}

func (s *DeploymentService) loadDeploymentForPipeline(id string) (string, bool) {
	logger := s.deploymentLogger(id)

	deployment, ok, err := s.GetDeployment(id)
	if err != nil {
		logger("could not load deployment before validation: %v", err)
		return "", false
	}
	if !ok {
		logger("could not find deployment before validation")
		return "", false
	}

	logger("validating repository URL")
	repoURL := validation.NormalizeGitHubRepoURL(deployment.RepoURL)
	if err := validation.ValidateGitHubRepoURL(repoURL); err != nil {
		s.failDeploymentStep(id, err, "repository URL validation failed")
		return "", false
	}

	logger("repository URL validation passed")
	return repoURL, true
}

func (s *DeploymentService) cloneDeploymentRepository(id string, repoURL string) (string, bool) {
	sourcePath, err := s.cloner.Clone(context.Background(), id, repoURL, s.deploymentLogger(id))
	if err != nil {
		s.failDeploymentStep(id, err, "repository clone failed")
		return "", false
	}

	if _, err := s.SaveDeploymentSourcePath(id, sourcePath); err != nil {
		s.failDeploymentStep(id, err, "could not save source path")
		return "", false
	}

	return sourcePath, true
}

func (s *DeploymentService) inspectDeploymentSource(id string, sourcePath string) (source.InspectionResult, bool) {
	logger := s.deploymentLogger(id)

	inspection := source.Inspect(sourcePath)
	if _, err := s.UpdateDeploymentSourceInsights(id, inspection); err != nil {
		s.failDeploymentStep(id, err, "could not save source inspection results")
		return source.InspectionResult{}, false
	}

	logger("detected project type: %s", inspection.ProjectType)
	if inspection.Framework != "" {
		logger("detected framework: %s", inspection.Framework)
	}
	if inspection.StartCommand != "" {
		logger("inferred start command: %s", inspection.StartCommand)
	}
	for _, hint := range inspection.Hints {
		logger("readiness hint: %s", hint)
	}

	return inspection, true
}

func (s *DeploymentService) buildDeploymentImage(id string, repoURL string, sourcePath string) (buildrunner.RailpackBuildResult, bool) {
	logger := s.deploymentLogger(id)

	if _, err := s.TransitionDeployment(id, models.StatusBuilding); err != nil {
		s.logPipelineError(id, err, "could not start build")
		return buildrunner.RailpackBuildResult{}, false
	}

	logger("status changed to building")
	imageTag := s.imageTagForDeployment(id)
	buildResult, err := s.builder.Build(
		context.Background(),
		sourcePath,
		imageTag,
		buildrunner.RailpackBuildOptions{CacheKey: railpackCacheKey(repoURL)},
		func(line string) {
			logger("railpack: %s", line)
		},
	)
	if err != nil {
		s.failDeploymentStep(id, err, "railpack build failed")
		return buildrunner.RailpackBuildResult{}, false
	}

	if _, err := s.SaveDeploymentImageTag(id, buildResult.ImageTag); err != nil {
		s.failDeploymentStep(id, err, "could not save image tag")
		return buildrunner.RailpackBuildResult{}, false
	}

	logger("railpack build succeeded with image %s", buildResult.ImageTag)
	return buildResult, true
}

func (s *DeploymentService) startDeploymentRuntime(id string, imageTag string, startCommand string) (dockerruntime.ContainerInfo, bool) {
	logger := s.deploymentLogger(id)

	if _, err := s.TransitionDeployment(id, models.StatusDeploying); err != nil {
		s.logPipelineError(id, err, "could not start deploy")
		return dockerruntime.ContainerInfo{}, false
	}

	logger("status changed to deploying")
	deployCtx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	envVars, err := s.store.GetEnvVars(id)
	if err != nil {
		s.failDeploymentStep(id, err, "could not load deployment environment variables")
		return dockerruntime.ContainerInfo{}, false
	}
	if len(envVars) > 0 {
		logger("docker: injecting environment variables: %s", strings.Join(sortedMapKeys(envVars), ", "))
	}

	runtimeInfo, err := s.runtime.Run(deployCtx, id, imageTag, startCommand, envVars, logger)
	if err != nil {
		s.failDeploymentStep(id, err, "docker runtime failed")
		return dockerruntime.ContainerInfo{}, false
	}

	if _, err := s.SaveDeploymentRuntime(id, runtimeInfo); err != nil {
		s.failDeploymentStepWithCleanup(id, func() {
			s.runtime.Cleanup(context.Background(), runtimeInfo.ContainerName)
		}, err, "could not save runtime metadata")
		return dockerruntime.ContainerInfo{}, false
	}

	return runtimeInfo, true
}

func (s *DeploymentService) configureDeploymentIngress(id string, runtimeInfo dockerruntime.ContainerInfo) bool {
	logger := s.deploymentLogger(id)

	routes, err := s.caddyRoutesWith(id, runtimeInfo.HostPort)
	if err != nil {
		s.failDeploymentStepWithCleanup(id, func() {
			s.runtime.Cleanup(context.Background(), runtimeInfo.ContainerName)
		}, err, "could not build caddy routes")
		return false
	}

	logger("caddy: configuring host route for %s", s.router.LiveURL(id))
	deployCtx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	if err := s.router.ApplyRoutes(deployCtx, routes, logger); err != nil {
		s.failDeploymentStepWithCleanupAndResync(id, func() {
			s.runtime.Cleanup(context.Background(), runtimeInfo.ContainerName)
		}, err, "caddy route update failed")
		return false
	}

	return true
}

func (s *DeploymentService) completeDeploymentRun(id string, imageTag string, containerName string) bool {
	liveURL := s.router.LiveURL(id)
	if _, err := s.CompleteDeployment(id, imageTag, liveURL); err != nil {
		s.failDeploymentStepWithCleanupAndResync(id, func() {
			s.runtime.Cleanup(context.Background(), containerName)
		}, err, "could not complete deployment")
		return false
	}

	return true
}
