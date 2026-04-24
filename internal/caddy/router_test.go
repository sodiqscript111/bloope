package caddy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRouterWritesHostBasedRoutes(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "Caddyfile")
	router := &Router{
		configPath:  configPath,
		ingressPort: 8081,
		hostSuffix:  "localhost",
		scheme:      "http",
	}

	if err := router.writeConfig([]Route{{DeploymentID: "dep_test", HostPort: 49152}}); err != nil {
		t.Fatalf("write config: %v", err)
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	text := string(content)
	for _, expected := range []string{
		"http://dep-test.localhost:8081",
		"reverse_proxy host.docker.internal:49152",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected Caddyfile to contain %q, got:\n%s", expected, text)
		}
	}
}

func TestRouterLiveURLUsesDeploymentHost(t *testing.T) {
	router := &Router{
		ingressPort: 8081,
		hostSuffix:  "localhost",
		scheme:      "http",
	}

	got := router.LiveURL("dep_test")
	want := "http://dep-test.localhost:8081"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
