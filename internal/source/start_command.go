package source

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type StartCommandResult struct {
	Framework string
	Command   string
	Notes     []string
}

func DetectStartCommand(root string) StartCommandResult {
	dependencies := strings.ToLower(readDependencyText(root))

	if strings.Contains(dependencies, "django") && fileExists(filepath.Join(root, "manage.py")) {
		if project := detectDjangoProject(root); project != "" {
			return StartCommandResult{
				Framework: "Django",
				Command:   "gunicorn " + project + ".wsgi:application --bind 0.0.0.0:${PORT:-8080}",
				Notes:     []string{"Detected Django start command from manage.py and wsgi.py"},
			}
		}
	}

	if strings.Contains(dependencies, "fastapi") && strings.Contains(dependencies, "uvicorn") {
		if module := findPythonAppModule(root, `(?m)\bapp\s*=\s*FastAPI\s*\(`); module != "" {
			return StartCommandResult{
				Framework: "FastAPI",
				Command:   "uvicorn " + module + ":app --host 0.0.0.0 --port ${PORT:-8080}",
				Notes:     []string{"Detected FastAPI start command from app = FastAPI(...)"},
			}
		}
	}

	if strings.Contains(dependencies, "flask") {
		if module := findPythonAppModule(root, `(?m)\bapp\s*=\s*Flask\s*\(`); module != "" {
			return StartCommandResult{
				Framework: "Flask",
				Command:   "gunicorn " + module + ":app --bind 0.0.0.0:${PORT:-8080}",
				Notes:     []string{"Detected Flask start command from app = Flask(...)"},
			}
		}
	}

	return StartCommandResult{}
}

func readDependencyText(root string) string {
	paths := []string{
		filepath.Join(root, "requirements.txt"),
		filepath.Join(root, "pyproject.toml"),
	}

	var builder strings.Builder
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err == nil {
			builder.Write(content)
			builder.WriteByte('\n')
		}
	}

	return builder.String()
}

func detectDjangoProject(root string) string {
	matches, err := filepath.Glob(filepath.Join(root, "*", "wsgi.py"))
	if err != nil || len(matches) == 0 {
		return ""
	}

	for _, match := range matches {
		project := filepath.Base(filepath.Dir(match))
		if fileExists(filepath.Join(root, project, "__init__.py")) {
			return project
		}
	}

	return ""
}

func findPythonAppModule(root string, pattern string) string {
	re := regexp.MustCompile(pattern)
	candidates := []string{
		filepath.Join(root, "main.py"),
		filepath.Join(root, "app.py"),
		filepath.Join(root, "src", "main.py"),
		filepath.Join(root, "src", "app.py"),
	}

	for _, candidate := range candidates {
		content, err := os.ReadFile(candidate)
		if err != nil || !re.Match(content) {
			continue
		}

		return pythonModulePath(root, candidate)
	}

	return ""
}

func pythonModulePath(root string, path string) string {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return ""
	}

	relative = strings.TrimSuffix(relative, filepath.Ext(relative))
	parts := strings.Split(relative, string(filepath.Separator))
	return strings.Join(parts, ".")
}
