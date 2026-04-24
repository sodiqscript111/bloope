package build

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRailpackBuilderStreamsOutput(t *testing.T) {
	commandPath := writeFakeRailpack(t, `echo stdout-line
echo stderr-line 1>&2
exit 0`)

	builder := &RailpackBuilder{
		binary:  commandPath,
		timeout: 5 * time.Second,
	}

	var lines []string
	result, err := builder.Build(context.Background(), t.TempDir(), "bloope/test:railpack", RailpackBuildOptions{}, func(line string) {
		lines = append(lines, line)
	})
	if err != nil {
		t.Fatalf("expected build to succeed: %v", err)
	}
	if result.ImageTag != "bloope/test:railpack" {
		t.Fatalf("expected image tag to be returned, got %q", result.ImageTag)
	}

	assertLineContains(t, lines, "stdout-line")
	assertLineContains(t, lines, "stderr-line")
}

func TestRailpackBuilderReturnsFailure(t *testing.T) {
	commandPath := writeFakeRailpack(t, `echo failed 1>&2
exit 7`)

	builder := &RailpackBuilder{
		binary:  commandPath,
		timeout: 5 * time.Second,
	}

	var lines []string
	_, err := builder.Build(context.Background(), t.TempDir(), "bloope/test:railpack", RailpackBuildOptions{}, func(line string) {
		lines = append(lines, line)
	})
	if err == nil {
		t.Fatal("expected build to fail")
	}
	assertLineContains(t, lines, "failed")
}

func TestRailpackBuilderBuildCommandIncludesCacheKey(t *testing.T) {
	builder := &RailpackBuilder{
		binary: "railpack",
	}

	command, err := builder.buildCommand(context.Background(), "app", "bloope/test:railpack", RailpackBuildOptions{
		CacheKey: "repo-123",
	})
	if err != nil {
		t.Fatalf("build command: %v", err)
	}

	args := strings.Join(command.Args, " ")
	if !strings.Contains(args, "--cache-key repo-123") {
		t.Fatalf("expected build command to include cache key, got %q", args)
	}
}

func TestShellQuoteEscapesSingleQuotes(t *testing.T) {
	got := shellQuote("owner's app")
	want := `'owner'"'"'s app'`
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func writeFakeRailpack(t *testing.T, body string) string {
	t.Helper()

	dir := t.TempDir()
	if runtime.GOOS == "windows" {
		path := filepath.Join(dir, "railpack.cmd")
		body = strings.ReplaceAll(body, "exit ", "exit /b ")
		content := "@echo off\r\n" + strings.ReplaceAll(body, "\n", "\r\n")
		if err := os.WriteFile(path, []byte(content), 0755); err != nil {
			t.Fatalf("write fake railpack command: %v", err)
		}
		return path
	}

	path := filepath.Join(dir, "railpack")
	content := "#!/bin/sh\n" + body + "\n"
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatalf("write fake railpack command: %v", err)
	}
	return path
}

func assertLineContains(t *testing.T, lines []string, expected string) {
	t.Helper()

	for _, line := range lines {
		if strings.Contains(line, expected) {
			return
		}
	}

	t.Fatalf("expected a line containing %q in %#v", expected, lines)
}
