package source

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectStartCommandFastAPI(t *testing.T) {
	root := t.TempDir()
	writeSourceFile(t, root, "requirements.txt", "fastapi\nuvicorn\n")
	writeSourceFile(t, root, filepath.Join("src", "main.py"), "from fastapi import FastAPI\napp = FastAPI()\n")

	result := DetectStartCommand(root)

	if result.Framework != "FastAPI" {
		t.Fatalf("expected FastAPI, got %q", result.Framework)
	}
	expected := "uvicorn src.main:app --host 0.0.0.0 --port ${PORT:-8080}"
	if result.Command != expected {
		t.Fatalf("expected %q, got %q", expected, result.Command)
	}
}

func TestDetectStartCommandFlask(t *testing.T) {
	root := t.TempDir()
	writeSourceFile(t, root, "requirements.txt", "flask\ngunicorn\n")
	writeSourceFile(t, root, "app.py", "from flask import Flask\napp = Flask(__name__)\n")

	result := DetectStartCommand(root)

	if result.Framework != "Flask" {
		t.Fatalf("expected Flask, got %q", result.Framework)
	}
	expected := "gunicorn app:app --bind 0.0.0.0:${PORT:-8080}"
	if result.Command != expected {
		t.Fatalf("expected %q, got %q", expected, result.Command)
	}
}

func TestDetectStartCommandDjango(t *testing.T) {
	root := t.TempDir()
	writeSourceFile(t, root, "requirements.txt", "django\ngunicorn\n")
	writeSourceFile(t, root, "manage.py", "# django manage\n")
	writeSourceFile(t, root, filepath.Join("chatwarden", "__init__.py"), "")
	writeSourceFile(t, root, filepath.Join("chatwarden", "wsgi.py"), "application = get_wsgi_application()\n")

	result := DetectStartCommand(root)

	if result.Framework != "Django" {
		t.Fatalf("expected Django, got %q", result.Framework)
	}
	expected := "gunicorn chatwarden.wsgi:application --bind 0.0.0.0:${PORT:-8080}"
	if result.Command != expected {
		t.Fatalf("expected %q, got %q", expected, result.Command)
	}
}

func writeSourceFile(t *testing.T, root string, name string, content string) {
	t.Helper()

	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("make directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}
