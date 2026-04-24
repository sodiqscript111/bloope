package build

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const DefaultRailpackTimeout = 30 * time.Minute

type RailpackBuilder struct {
	binary    string
	timeout   time.Duration
	useWSL    bool
	wslDistro string
}

type RailpackBuildResult struct {
	ImageTag string
}

type RailpackBuildOptions struct {
	CacheKey string
}

func NewRailpackBuilder() *RailpackBuilder {
	binary := railpackBinary()
	wslDistro := railpackWSLDistro()
	useWSL := shouldUseWSL(binary, wslDistro)

	return &RailpackBuilder{
		binary:    binary,
		timeout:   DefaultRailpackTimeout,
		useWSL:    useWSL,
		wslDistro: wslDistro,
	}
}

func railpackBinary() string {
	binary := strings.TrimSpace(os.Getenv("RAILPACK_BIN"))
	if binary == "" {
		binary = "railpack"
	}

	return binary
}

func railpackWSLDistro() string {
	distro := strings.TrimSpace(os.Getenv("RAILPACK_WSL_DISTRO"))
	if distro == "" {
		distro = "Ubuntu"
	}

	return distro
}

func shouldUseWSL(binary string, distro string) bool {
	choice := strings.TrimSpace(os.Getenv("RAILPACK_USE_WSL"))
	if choice != "" {
		return isTruthy(choice)
	}
	if runtime.GOOS != "windows" || strings.TrimSpace(os.Getenv("RAILPACK_BIN")) != "" {
		return false
	}

	return wslRailpackAvailable(distro)
}

func (b *RailpackBuilder) Build(ctx context.Context, sourceDir string, imageTag string, options RailpackBuildOptions, logLine func(string)) (RailpackBuildResult, error) {
	if b.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, b.timeout)
		defer cancel()
	}

	command, err := b.buildCommand(ctx, sourceDir, imageTag, options)
	if err != nil {
		return RailpackBuildResult{}, err
	}
	logLine(b.runnerDescription())
	if cacheKey := strings.TrimSpace(options.CacheKey); cacheKey != "" {
		logLine(fmt.Sprintf("using cache key %s", cacheKey))
	}

	stdout, err := command.StdoutPipe()
	if err != nil {
		return RailpackBuildResult{}, fmt.Errorf("open railpack stdout: %w", err)
	}

	stderr, err := command.StderrPipe()
	if err != nil {
		return RailpackBuildResult{}, fmt.Errorf("open railpack stderr: %w", err)
	}

	if err := command.Start(); err != nil {
		return RailpackBuildResult{}, fmt.Errorf("start railpack build: %w", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go scanOutput(&wg, stdout, logLine)
	go scanOutput(&wg, stderr, logLine)
	wg.Wait()

	err = command.Wait()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return RailpackBuildResult{}, fmt.Errorf("railpack build timed out after %s", b.timeout)
	}
	if err != nil {
		return RailpackBuildResult{}, fmt.Errorf("railpack build failed: %w", err)
	}

	return RailpackBuildResult{ImageTag: imageTag}, nil
}

func (b *RailpackBuilder) buildCommand(ctx context.Context, sourceDir string, imageTag string, options RailpackBuildOptions) (*exec.Cmd, error) {
	cacheKey := strings.TrimSpace(options.CacheKey)
	if !b.useWSL {
		args := []string{"build", "--name", imageTag, "--progress", "plain"}
		if cacheKey != "" {
			args = append(args, "--cache-key", cacheKey)
		}
		args = append(args, sourceDir)
		return exec.CommandContext(ctx, b.binary, args...), nil
	}

	wslSourceDir, err := windowsPathToWSL(sourceDir)
	if err != nil {
		return nil, err
	}

	commandLine := fmt.Sprintf(
		"%s$HOME/.local/bin/railpack build --name %s --progress plain%s %s",
		wslEnvironmentPrefix(),
		shellQuote(imageTag),
		wslCacheKeyArg(cacheKey),
		shellQuote(wslSourceDir),
	)
	args := []string{}
	if b.wslDistro != "" {
		args = append(args, "-d", b.wslDistro)
	}
	args = append(args, "--", "bash", "-lc", commandLine)

	return exec.CommandContext(ctx, "wsl.exe", args...), nil
}

func (b *RailpackBuilder) runnerDescription() string {
	if b.useWSL {
		return fmt.Sprintf("railpack runner: WSL distro %s", b.wslDistro)
	}

	return fmt.Sprintf("railpack runner: native binary %s", b.binary)
}

func wslRailpackAvailable(distro string) bool {
	args := []string{}
	if distro != "" {
		args = append(args, "-d", distro)
	}
	args = append(args, "--", "bash", "-lc", "test -x ~/.local/bin/railpack")

	return exec.Command("wsl.exe", args...).Run() == nil
}

func wslEnvironmentPrefix() string {
	buildkitHost := strings.TrimSpace(os.Getenv("BUILDKIT_HOST"))
	if buildkitHost == "" {
		return ""
	}

	return "BUILDKIT_HOST=" + shellQuote(buildkitHost) + " "
}

func wslCacheKeyArg(cacheKey string) string {
	if strings.TrimSpace(cacheKey) == "" {
		return ""
	}

	return " --cache-key " + shellQuote(cacheKey)
}

func scanOutput(wg *sync.WaitGroup, reader io.Reader, logLine func(string)) {
	defer wg.Done()

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		logLine(line)
	}
	if err := scanner.Err(); err != nil {
		logLine(fmt.Sprintf("error reading railpack output: %v", err))
	}
}

func windowsPathToWSL(path string) (string, error) {
	if runtime.GOOS != "windows" {
		return path, nil
	}

	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve source path: %w", err)
	}

	volume := filepath.VolumeName(absolutePath)
	if len(volume) < 2 || volume[1] != ':' {
		return "", fmt.Errorf("cannot convert path %q to WSL path", absolutePath)
	}

	drive := strings.ToLower(volume[:1])
	rest := strings.TrimPrefix(absolutePath, volume)
	rest = strings.ReplaceAll(rest, `\`, "/")
	return "/mnt/" + drive + rest, nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func isTruthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
