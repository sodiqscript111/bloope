package repository

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Cloner struct {
	gitBinary   string
	basePath    string
	githubToken string
}

func NewCloner() *Cloner {
	return &Cloner{
		gitBinary:   envOrDefault("BLOOPE_GIT_BIN", "git"),
		basePath:    filepath.Join("tmp", "deployments"),
		githubToken: strings.TrimSpace(os.Getenv("BLOOPE_GITHUB_TOKEN")),
	}
}

func (c *Cloner) Clone(ctx context.Context, deploymentID string, repoURL string, logLine func(string, ...any)) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	sourcePath := filepath.Join(c.basePath, deploymentID, "source")
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0755); err != nil {
		return "", fmt.Errorf("prepare clone directory: %w", err)
	}
	if err := os.RemoveAll(sourcePath); err != nil {
		return "", fmt.Errorf("clear source directory: %w", err)
	}

	logLine("cloning repository into %s", sourcePath)

	command := c.gitCloneCommand(ctx, repoURL, sourcePath)
	output, err := command.CombinedOutput()
	outputText := sanitizeGitOutput(string(output), c.githubToken)
	if strings.TrimSpace(outputText) != "" {
		logLine("git clone output: %s", strings.TrimSpace(outputText))
	}
	if err != nil {
		return "", gitCloneError(outputText, err, c.githubToken != "")
	}

	logLine("repository cloned successfully")
	return sourcePath, nil
}

func (c *Cloner) gitCloneCommand(ctx context.Context, repoURL string, sourcePath string) *exec.Cmd {
	args := []string{}
	if c.githubToken != "" {
		credentials := base64.StdEncoding.EncodeToString([]byte("x-access-token:" + c.githubToken))
		args = append(args, "-c", "http.https://github.com/.extraheader=Authorization: Basic "+credentials)
	}
	args = append(args, "clone", "--depth", "1", repoURL, sourcePath)

	command := exec.CommandContext(ctx, c.gitBinary, args...)
	command.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GCM_INTERACTIVE=Never")
	return command
}

func gitCloneError(outputText string, err error, tokenConfigured bool) error {
	loweredOutput := strings.ToLower(outputText)
	if strings.Contains(loweredOutput, "could not read username") ||
		strings.Contains(loweredOutput, "authentication failed") ||
		strings.Contains(loweredOutput, "repository not found") {
		if tokenConfigured {
			return fmt.Errorf("git clone failed: repository is private, missing, or BLOOPE_GITHUB_TOKEN cannot access it: %w", err)
		}

		return fmt.Errorf("git clone failed: repository is private or missing; make it public or set BLOOPE_GITHUB_TOKEN: %w", err)
	}

	return fmt.Errorf("git clone failed: %w", err)
}

func sanitizeGitOutput(output string, githubToken string) string {
	if githubToken == "" {
		return output
	}

	credentials := base64.StdEncoding.EncodeToString([]byte("x-access-token:" + githubToken))
	output = strings.ReplaceAll(output, githubToken, "[redacted]")
	output = strings.ReplaceAll(output, credentials, "[redacted]")
	return output
}

func envOrDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}
