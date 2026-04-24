package docker

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestContainerNameForDeployment(t *testing.T) {
	got := ContainerNameForDeployment("dep_123")
	if got != "bloope-dep_123" {
		t.Fatalf("unexpected container name %q", got)
	}
}

func TestProbeHealthTargetsAcceptsClientErrorsAndEventuallySucceeds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.NotFound(writer, request)
	}))
	defer server.Close()

	result, err := probeHealthTargets(context.Background(), server.Client(), []string{
		"http://127.0.0.1:1/healthz",
		server.URL + "/",
	})
	if err != nil {
		t.Fatalf("expected health probe to succeed: %v", err)
	}
	if result.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 to count as healthy, got %d", result.StatusCode)
	}
	if result.URL != server.URL+"/" {
		t.Fatalf("expected successful probe url to round-trip, got %q", result.URL)
	}
}

func TestProbeHealthTargetsRejectsServerErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Error(writer, "not ready", http.StatusBadGateway)
	}))
	defer server.Close()

	_, err := probeHealthTargets(context.Background(), server.Client(), []string{server.URL + "/"})
	if err == nil {
		t.Fatal("expected health probe failure for 5xx response")
	}
	expected := fmt.Sprintf("%s/ returned status %d", server.URL, http.StatusBadGateway)
	if err.Error() != expected {
		t.Fatalf("expected %q, got %q", expected, err.Error())
	}
}

func TestEnvDurationOrDefaultFallsBackForInvalidValues(t *testing.T) {
	t.Setenv("BLOOPE_HEALTHCHECK_TIMEOUT", "not-a-duration")
	if got := envDurationOrDefault("BLOOPE_HEALTHCHECK_TIMEOUT", 5*time.Second); got != 5*time.Second {
		t.Fatalf("expected fallback duration, got %s", got)
	}
}
