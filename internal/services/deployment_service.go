package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	buildrunner "bloope/internal/build"
	caddyrouter "bloope/internal/caddy"
	dockerruntime "bloope/internal/docker"
	"bloope/internal/models"
	repository "bloope/internal/repository"
	"bloope/internal/source"
	"bloope/internal/state"
	"bloope/internal/validation"
)

var envVarNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type DeploymentService struct {
	store     *DeploymentStore
	engine    *state.Engine
	logBroker *DeploymentLogBroker
	cloner    *repository.Cloner
	builder   *buildrunner.RailpackBuilder
	runtime   *dockerruntime.Runtime
	router    *caddyrouter.Router
	nextID    atomic.Uint64
}

func NewDeploymentService(store *DeploymentStore) *DeploymentService {
	return &DeploymentService{
		store:     store,
		engine:    state.NewEngine(),
		logBroker: NewDeploymentLogBroker(),
		cloner:    repository.NewCloner(),
		builder:   buildrunner.NewRailpackBuilder(),
		runtime:   dockerruntime.NewRuntime(),
		router:    caddyrouter.NewRouter(),
	}
}

func (s *DeploymentService) CreateDeployment(repoURL string, envVars map[string]string) (*models.Deployment, error) {
	repoURL = strings.TrimSpace(repoURL)
	if repoURL == "" {
		return nil, fmt.Errorf("repo_url is required")
	}
	normalizedEnvVars, err := normalizeEnvVars(envVars)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	id := s.generateID()
	deployment := &models.Deployment{
		ID:             id,
		RepoURL:        repoURL,
		Status:         models.StatusPending,
		EnvVars:        normalizedEnvVars,
		ReadinessHints: []string{},
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	created, err := s.store.Create(deployment)
	if err != nil {
		return nil, err
	}
	go s.runFakeDeployment(created.ID)

	return created, nil
}

func (s *DeploymentService) ListDeployments() ([]*models.Deployment, error) {
	return s.store.List()
}

func (s *DeploymentService) GetDeployment(id string) (*models.Deployment, bool, error) {
	return s.store.GetByID(id)
}

func (s *DeploymentService) GetDeploymentLogs(id string) ([]*models.DeploymentLog, bool, error) {
	return s.store.GetLogsByDeploymentID(id)
}

func (s *DeploymentService) SubscribeDeploymentLogs(id string) (<-chan *models.DeploymentLog, func(), bool) {
	if _, ok, err := s.store.GetByID(id); err != nil || !ok {
		return nil, nil, false
	}

	logs, unsubscribe := s.logBroker.Subscribe(id)
	return logs, unsubscribe, true
}

func (s *DeploymentService) TransitionDeployment(id string, nextStatus models.DeploymentStatus) (*models.Deployment, error) {
	deployment, ok, err := s.store.GetByID(id)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrDeploymentNotFound
	}

	if err := s.engine.ValidateTransition(deployment.Status, nextStatus); err != nil {
		return nil, err
	}

	return s.store.SetStatus(id, nextStatus)
}

func (s *DeploymentService) CompleteDeployment(id string, imageTag string, liveURL string) (*models.Deployment, error) {
	deployment, ok, err := s.store.GetByID(id)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrDeploymentNotFound
	}

	if err := s.engine.ValidateTransition(deployment.Status, models.StatusRunning); err != nil {
		return nil, err
	}

	return s.store.Complete(id, imageTag, liveURL)
}

func (s *DeploymentService) SaveDeploymentImageTag(id string, imageTag string) (*models.Deployment, error) {
	return s.store.SaveImageTag(id, imageTag)
}

func (s *DeploymentService) SaveDeploymentSourcePath(id string, sourcePath string) (*models.Deployment, error) {
	return s.store.SaveSourcePath(id, sourcePath)
}

func (s *DeploymentService) SaveDeploymentRuntime(id string, runtime dockerruntime.ContainerInfo) (*models.Deployment, error) {
	return s.store.SaveRuntime(id, runtime.ContainerName, runtime.ContainerID, runtime.HostPort)
}

func (s *DeploymentService) FailDeployment(id string, message string) (*models.Deployment, error) {
	deployment, ok, err := s.store.GetByID(id)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrDeploymentNotFound
	}

	if err := s.engine.ValidateTransition(deployment.Status, models.StatusFailed); err != nil && deployment.Status != models.StatusFailed {
		return nil, err
	}

	return s.store.Fail(id, message)
}

func (s *DeploymentService) UpdateDeploymentSourceInsights(id string, inspection source.InspectionResult) (*models.Deployment, error) {
	return s.store.SaveSourceInsights(id, inspection.ProjectType, inspection.Framework, inspection.StartCommand, inspection.Hints)
}

func (s *DeploymentService) SyncCaddyRoutes(ctx context.Context) error {
	routes, err := s.caddyRoutes()
	if err != nil {
		return err
	}
	for _, route := range routes {
		if _, err := s.store.SaveLiveURL(route.DeploymentID, s.router.LiveURL(route.DeploymentID)); err != nil {
			return err
		}
	}

	return s.router.ApplyRoutes(ctx, routes, func(format string, args ...any) {
		log.Printf("[caddy] "+format, args...)
	})
}

func (s *DeploymentService) generateID() string {
	return fmt.Sprintf("dep_%d_%06d", time.Now().UTC().UnixMilli(), s.nextID.Add(1))
}

func (s *DeploymentService) imageTagForDeployment(id string) string {
	return fmt.Sprintf("bloope/%s:railpack", id)
}

func railpackCacheKey(repoURL string) string {
	normalized := validation.NormalizeGitHubRepoURL(repoURL)
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(normalized))))
	return "repo-" + hex.EncodeToString(sum[:8])
}

func normalizeEnvVars(envVars map[string]string) (map[string]string, error) {
	if len(envVars) == 0 {
		return nil, nil
	}

	normalized := make(map[string]string, len(envVars))
	for key, value := range envVars {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if !envVarNamePattern.MatchString(key) {
			return nil, fmt.Errorf("invalid environment variable name %q", key)
		}
		normalized[key] = value
	}

	return normalized, nil
}

func sortedMapKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (s *DeploymentService) caddyRoutes() ([]caddyrouter.Route, error) {
	deployments, err := s.store.ListRunning()
	if err != nil {
		return nil, err
	}

	routes := make([]caddyrouter.Route, 0, len(deployments))
	for _, deployment := range deployments {
		if deployment.HostPort <= 0 {
			continue
		}
		routes = append(routes, caddyrouter.Route{
			DeploymentID: deployment.ID,
			HostPort:     deployment.HostPort,
		})
	}

	return routes, nil
}

func (s *DeploymentService) caddyRoutesWith(deploymentID string, hostPort int) ([]caddyrouter.Route, error) {
	routes, err := s.caddyRoutes()
	if err != nil {
		return nil, err
	}

	found := false
	for index := range routes {
		if routes[index].DeploymentID == deploymentID {
			routes[index].HostPort = hostPort
			found = true
			break
		}
	}
	if !found {
		routes = append(routes, caddyrouter.Route{
			DeploymentID: deploymentID,
			HostPort:     hostPort,
		})
	}

	return routes, nil
}

func (s *DeploymentService) syncCaddyRoutesBestEffort(id string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.SyncCaddyRoutes(ctx); err != nil {
		s.logDeployment(id, "could not resync caddy routes after failure: %v", err)
	}
}
