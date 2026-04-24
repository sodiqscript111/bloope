package source

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInspectDetectsGoProjectAndHints(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example\n")
	writeFile(t, root, "Dockerfile", "FROM golang:1.25\n")
	writeFile(t, root, "config.go", "db.SetMaxOpenConns(150)\naddr := \"localhost:8080\"\n")

	result := Inspect(root)

	if result.ProjectType != ProjectTypeGo {
		t.Fatalf("expected project type %q, got %q", ProjectTypeGo, result.ProjectType)
	}
	assertContains(t, result.Hints, "Detected Go project via go.mod")
	assertContains(t, result.Hints, "Dockerfile found but pipeline is expected to use Railpack later")
	assertContains(t, result.Hints, "Found hardcoded localhost reference")
	assertContains(t, result.Hints, "Suspicious DB pool config might be high")
}

func TestInspectDetectsUnknownProject(t *testing.T) {
	result := Inspect(t.TempDir())

	if result.ProjectType != ProjectTypeUnknown {
		t.Fatalf("expected project type %q, got %q", ProjectTypeUnknown, result.ProjectType)
	}
	assertContains(t, result.Hints, "No obvious supported app manifest found")
}

func writeFile(t *testing.T, root string, name string, content string) {
	t.Helper()

	path := filepath.Join(root, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
}

func assertContains(t *testing.T, values []string, expected string) {
	t.Helper()

	for _, value := range values {
		if value == expected {
			return
		}
	}

	t.Fatalf("expected %q in %#v", expected, values)
}
