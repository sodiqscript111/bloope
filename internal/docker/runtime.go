package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

const defaultContainerPort = 8080

var defaultHealthCheckPaths = []string{"/healthz", "/health", "/readyz", "/ready", "/"}

type Runtime struct {
	binary               string
	defaultContainerPort int
	healthCheckTimeout   time.Duration
	healthCheckInterval  time.Duration
}

type ContainerInfo struct {
	ContainerName string
	ContainerID   string
	HostPort      int
	ContainerPort int
}

type HealthCheckResult struct {
	URL        string
	StatusCode int
}

func NewRuntime() *Runtime {
	binary := strings.TrimSpace(os.Getenv("DOCKER_BIN"))
	if binary == "" {
		binary = "docker"
	}

	containerPort := defaultContainerPort
	if configuredPort, err := strconv.Atoi(strings.TrimSpace(os.Getenv("BLOOPE_CONTAINER_PORT"))); err == nil && configuredPort > 0 {
		containerPort = configuredPort
	}

	return &Runtime{
		binary:               binary,
		defaultContainerPort: containerPort,
		healthCheckTimeout:   envDurationOrDefault("BLOOPE_HEALTHCHECK_TIMEOUT", 45*time.Second),
		healthCheckInterval:  envDurationOrDefault("BLOOPE_HEALTHCHECK_INTERVAL", 1*time.Second),
	}
}

func (r *Runtime) Run(ctx context.Context, deploymentID string, imageTag string, startCommand string, envVars map[string]string, logLine func(string, ...any)) (ContainerInfo, error) {
	containerName := ContainerNameForDeployment(deploymentID)

	logLine("docker: removing any existing container named %s", containerName)
	if output, err := r.command(ctx, "rm", "-f", containerName).CombinedOutput(); err != nil && len(output) > 0 {
		logLine("docker: cleanup output: %s", strings.TrimSpace(string(output)))
	}

	containerPort := r.detectContainerPort(ctx, imageTag, logLine)
	hostPort, err := findFreePort()
	if err != nil {
		return ContainerInfo{}, err
	}

	logLine("docker: starting %s from image %s on host port %d -> container port %d", containerName, imageTag, hostPort, containerPort)
	args := []string{
		"run",
		"-d",
		"--name", containerName,
		"--add-host", "host.docker.internal:host-gateway",
		"-e", fmt.Sprintf("PORT=%d", containerPort),
		"-p", fmt.Sprintf("0.0.0.0:%d:%d", hostPort, containerPort),
	}
	for _, key := range sortedEnvKeys(envVars) {
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, envVars[key]))
	}
	if strings.TrimSpace(startCommand) != "" {
		// Railpack images commonly use a shell entrypoint. Override it so our
		// detected command is executed as the shell command, not as bash's $0.
		args = append(args, "--entrypoint", "sh")
	}
	args = append(args, imageTag)
	if strings.TrimSpace(startCommand) != "" {
		logLine("docker: overriding image start command with: %s", startCommand)
		args = append(args, "-c", startCommand)
	}

	output, err := r.command(ctx, args...).CombinedOutput()
	if len(output) > 0 {
		logLine("docker: run output: %s", strings.TrimSpace(string(output)))
	}
	if err != nil {
		return ContainerInfo{}, fmt.Errorf("docker run failed: %w", err)
	}

	containerID := strings.TrimSpace(string(output))
	if containerID == "" {
		containerID = containerName
	}

	logLine("docker: waiting for health check on host port %d", hostPort)
	healthResult, err := r.waitForHealthy(ctx, containerName, hostPort)
	if err != nil {
		r.captureStartupFailure(ctx, containerName, logLine)
		return ContainerInfo{}, err
	}

	logLine("docker: health check passed at %s with status %d", healthResult.URL, healthResult.StatusCode)
	return ContainerInfo{
		ContainerName: containerName,
		ContainerID:   containerID,
		HostPort:      hostPort,
		ContainerPort: containerPort,
	}, nil
}

func (r *Runtime) Cleanup(ctx context.Context, containerName string) {
	if strings.TrimSpace(containerName) == "" {
		return
	}
	_, _ = r.command(ctx, "rm", "-f", containerName).CombinedOutput()
}

func ContainerNameForDeployment(deploymentID string) string {
	return "bloope-" + deploymentID
}

func (r *Runtime) detectContainerPort(ctx context.Context, imageTag string, logLine func(string, ...any)) int {
	output, err := r.command(ctx, "image", "inspect", imageTag, "--format", "{{json .Config.ExposedPorts}}").CombinedOutput()
	if err != nil {
		logLine("docker: could not inspect exposed ports, using default %d: %v", r.defaultContainerPort, err)
		return r.defaultContainerPort
	}

	var exposed map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(output))), &exposed); err != nil || len(exposed) == 0 {
		logLine("docker: no exposed ports found, using MVP default container port %d", r.defaultContainerPort)
		return r.defaultContainerPort
	}

	ports := make([]int, 0, len(exposed))
	for rawPort := range exposed {
		portText, _, ok := strings.Cut(rawPort, "/")
		if !ok {
			portText = rawPort
		}
		port, err := strconv.Atoi(portText)
		if err == nil && port > 0 {
			ports = append(ports, port)
		}
	}
	if len(ports) == 0 {
		return r.defaultContainerPort
	}

	sort.Ints(ports)
	logLine("docker: using exposed image port %d", ports[0])
	return ports[0]
}

func (r *Runtime) isContainerRunning(ctx context.Context, containerName string) (bool, error) {
	output, err := r.command(ctx, "inspect", "-f", "{{.State.Running}}", containerName).CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("inspect container state: %w", err)
	}

	return strings.TrimSpace(string(output)) == "true", nil
}

func (r *Runtime) waitForHealthy(parent context.Context, containerName string, hostPort int) (HealthCheckResult, error) {
	ctx := parent
	cancel := func() {}
	if r.healthCheckTimeout > 0 {
		ctx, cancel = context.WithTimeout(parent, r.healthCheckTimeout)
	}
	defer cancel()

	client := &http.Client{Timeout: 3 * time.Second}
	targets := healthCheckTargets(hostPort)
	ticker := time.NewTicker(r.healthCheckInterval)
	defer ticker.Stop()

	var lastErr error
	for {
		running, err := r.isContainerRunning(ctx, containerName)
		if err != nil {
			lastErr = err
		} else if !running {
			return HealthCheckResult{}, fmt.Errorf("container %s exited during health checks", containerName)
		} else {
			result, err := probeHealthTargets(ctx, client, targets)
			if err == nil {
				return result, nil
			}
			lastErr = err
		}

		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) && lastErr != nil {
				return HealthCheckResult{}, fmt.Errorf("health check timed out: %w", lastErr)
			}
			if lastErr != nil {
				return HealthCheckResult{}, lastErr
			}
			return HealthCheckResult{}, ctx.Err()
		case <-ticker.C:
		}
	}
}

func probeHealthTargets(ctx context.Context, client *http.Client, targets []string) (HealthCheckResult, error) {
	var lastErr error
	for _, target := range targets {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
		if err != nil {
			lastErr = fmt.Errorf("build health check request for %s: %w", target, err)
			continue
		}

		response, err := client.Do(request)
		if err != nil {
			lastErr = fmt.Errorf("%s: %w", target, err)
			continue
		}

		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 512))
		response.Body.Close()

		if isHealthyHTTPStatus(response.StatusCode) {
			return HealthCheckResult{
				URL:        target,
				StatusCode: response.StatusCode,
			}, nil
		}

		lastErr = fmt.Errorf("%s returned status %d", target, response.StatusCode)
	}

	if lastErr == nil {
		lastErr = errors.New("no health check targets configured")
	}
	return HealthCheckResult{}, lastErr
}

func isHealthyHTTPStatus(statusCode int) bool {
	return statusCode >= 200 && statusCode < 500
}

func healthCheckTargets(hostPort int) []string {
	targets := make([]string, 0, len(defaultHealthCheckPaths)*3)
	for _, host := range orderedHealthCheckHosts() {
		for _, path := range defaultHealthCheckPaths {
			targets = append(targets, fmt.Sprintf("http://%s:%d%s", host, hostPort, path))
		}
	}

	return targets
}

func orderedHealthCheckHosts() []string {
	candidates := []string{"127.0.0.1", "localhost", "host.docker.internal"}
	if runningInContainer() {
		candidates = []string{"host.docker.internal", "127.0.0.1", "localhost"}
	}

	seen := map[string]struct{}{}
	hosts := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		hosts = append(hosts, candidate)
	}

	return hosts
}

func runningInContainer() bool {
	_, err := os.Stat("/.dockerenv")
	return err == nil
}

func (r *Runtime) captureStartupFailure(ctx context.Context, containerName string, logLine func(string, ...any)) {
	logCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if logs, logErr := r.command(logCtx, "logs", "--tail", "80", containerName).CombinedOutput(); logErr == nil && len(logs) > 0 {
		logLine("docker: container logs: %s", strings.TrimSpace(string(logs)))
	}
	_, _ = r.command(logCtx, "rm", "-f", containerName).CombinedOutput()
}

func (r *Runtime) command(ctx context.Context, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, r.binary, args...)
}

func findFreePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("find free host port: %w", err)
	}
	defer listener.Close()

	address := listener.Addr().(*net.TCPAddr)
	return address.Port, nil
}

func sortedEnvKeys(envVars map[string]string) []string {
	keys := make([]string, 0, len(envVars))
	for key := range envVars {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func envDurationOrDefault(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return fallback
	}

	return duration
}
