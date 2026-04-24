package caddy

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	defaultIngressPort = 8081
	defaultContainer   = "bloope-caddy"
)

type Route struct {
	DeploymentID string
	HostPort     int
}

type Router struct {
	dockerBinary  string
	containerName string
	configPath    string
	ingressPort   int
	hostSuffix    string
	scheme        string
}

func NewRouter() *Router {
	dockerBinary := strings.TrimSpace(os.Getenv("DOCKER_BIN"))
	if dockerBinary == "" {
		dockerBinary = "docker"
	}

	configPath := strings.TrimSpace(os.Getenv("BLOOPE_CADDYFILE_PATH"))
	if configPath == "" {
		configPath = filepath.Join("tmp", "caddy", "Caddyfile")
	}

	ingressPort := defaultIngressPort
	if configuredPort, err := strconv.Atoi(strings.TrimSpace(os.Getenv("BLOOPE_CADDY_PORT"))); err == nil && configuredPort > 0 {
		ingressPort = configuredPort
	}

	hostSuffix := strings.Trim(strings.TrimSpace(os.Getenv("BLOOPE_DEPLOYMENT_HOST_SUFFIX")), ".")
	if hostSuffix == "" {
		hostSuffix = "localhost"
	}

	scheme := strings.TrimSpace(os.Getenv("BLOOPE_PUBLIC_SCHEME"))
	if scheme == "" {
		scheme = "http"
	}

	return &Router{
		dockerBinary:  dockerBinary,
		containerName: envOrDefault("BLOOPE_CADDY_CONTAINER", defaultContainer),
		configPath:    configPath,
		ingressPort:   ingressPort,
		hostSuffix:    hostSuffix,
		scheme:        scheme,
	}
}

func (r *Router) ApplyRoutes(ctx context.Context, routes []Route, logLine func(string, ...any)) error {
	if err := r.writeConfig(routes); err != nil {
		return err
	}

	if err := r.ensureContainer(ctx, logLine); err != nil {
		return err
	}

	logLine("caddy: reloading routes")
	output, err := r.command(ctx, "exec", r.containerName, "caddy", "reload", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile").CombinedOutput()
	if len(output) > 0 {
		logLine("caddy: reload output: %s", strings.TrimSpace(string(output)))
	}
	if err != nil {
		return fmt.Errorf("reload caddy: %w", err)
	}

	return nil
}

func (r *Router) LiveURL(deploymentID string) string {
	return fmt.Sprintf("%s://%s", r.scheme, r.hostForDeployment(deploymentID))
}

func (r *Router) writeConfig(routes []Route) error {
	if err := os.MkdirAll(filepath.Dir(r.configPath), 0755); err != nil {
		return fmt.Errorf("create caddy config directory: %w", err)
	}

	sort.Slice(routes, func(i, j int) bool {
		return routes[i].DeploymentID < routes[j].DeploymentID
	})

	var builder strings.Builder
	builder.WriteString("{\n\tauto_https off\n}\n\n")
	builder.WriteString(fmt.Sprintf("http://:%d {\n\trespond /healthz \"ok\" 200\n}\n", r.ingressPort))
	for _, route := range routes {
		if route.DeploymentID == "" || route.HostPort <= 0 {
			continue
		}
		builder.WriteString(fmt.Sprintf("\nhttp://%s {\n", r.hostForDeployment(route.DeploymentID)))
		builder.WriteString(fmt.Sprintf("\treverse_proxy host.docker.internal:%d\n", route.HostPort))
		builder.WriteString("}\n")
	}

	if err := os.WriteFile(r.configPath, []byte(builder.String()), 0644); err != nil {
		return fmt.Errorf("write caddy config: %w", err)
	}

	return nil
}

func (r *Router) hostForDeployment(deploymentID string) string {
	host := fmt.Sprintf("%s.%s", sanitizeHostLabel(deploymentID), r.hostSuffix)
	if r.ingressPort == 80 {
		return host
	}

	return fmt.Sprintf("%s:%d", host, r.ingressPort)
}

func (r *Router) ensureContainer(ctx context.Context, logLine func(string, ...any)) error {
	exists := r.command(ctx, "inspect", r.containerName).Run() == nil
	if exists {
		runningOutput, err := r.command(ctx, "inspect", "-f", "{{.State.Running}}", r.containerName).CombinedOutput()
		if err == nil && strings.TrimSpace(string(runningOutput)) == "true" {
			return nil
		}

		logLine("caddy: starting existing container %s", r.containerName)
		if output, err := r.command(ctx, "start", r.containerName).CombinedOutput(); err != nil {
			if len(output) > 0 {
				logLine("caddy: start output: %s", strings.TrimSpace(string(output)))
			}
			return fmt.Errorf("start caddy container: %w", err)
		}
		return nil
	}

	absoluteConfigPath, err := filepath.Abs(r.configPath)
	if err != nil {
		return fmt.Errorf("resolve caddy config path: %w", err)
	}

	logLine("caddy: starting container %s on port %d", r.containerName, r.ingressPort)
	output, err := r.command(
		ctx,
		"run",
		"-d",
		"--name", r.containerName,
		"--add-host", "host.docker.internal:host-gateway",
		"-p", fmt.Sprintf("%d:%d", r.ingressPort, r.ingressPort),
		"-v", fmt.Sprintf("%s:/etc/caddy/Caddyfile", absoluteConfigPath),
		"caddy:2-alpine",
		"caddy", "run", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile",
	).CombinedOutput()
	if len(output) > 0 {
		logLine("caddy: run output: %s", strings.TrimSpace(string(output)))
	}
	if err != nil {
		return fmt.Errorf("start caddy container: %w", err)
	}

	return nil
}

func (r *Router) command(ctx context.Context, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, r.dockerBinary, args...)
}

func envOrDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}

func sanitizeHostLabel(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	previousDash := false

	for _, char := range value {
		valid := (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9')
		if valid {
			builder.WriteRune(char)
			previousDash = false
			continue
		}
		if !previousDash {
			builder.WriteRune('-')
			previousDash = true
		}
	}

	label := strings.Trim(builder.String(), "-")
	if label == "" {
		return "deployment"
	}

	if len(label) > 63 {
		label = strings.TrimRight(label[:63], "-")
	}

	return label
}
