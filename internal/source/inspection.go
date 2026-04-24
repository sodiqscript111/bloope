package source

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	ProjectTypeGo      = "Go"
	ProjectTypeNode    = "Node"
	ProjectTypePython  = "Python"
	ProjectTypeRust    = "Rust"
	ProjectTypeUnknown = "Unknown"
)

const (
	maxScannedFiles    = 60
	maxScannedFileSize = 256 * 1024
)

type InspectionResult struct {
	ProjectType  string
	Framework    string
	StartCommand string
	Notes        []string
	Hints        []string
}

func Inspect(root string) InspectionResult {
	result := InspectionResult{
		ProjectType: ProjectTypeUnknown,
	}

	result.ProjectType, result.Notes = detectProjectType(root)
	result.Hints = append(result.Hints, result.Notes...)

	startCommand := DetectStartCommand(root)
	result.Framework = startCommand.Framework
	result.StartCommand = startCommand.Command
	for _, note := range startCommand.Notes {
		result.Hints = append(result.Hints, note)
	}

	if result.ProjectType == ProjectTypeUnknown {
		result.Hints = append(result.Hints, "No obvious supported app manifest found")
	}

	if fileExists(filepath.Join(root, "Dockerfile")) {
		result.Hints = append(result.Hints, "Dockerfile found but pipeline is expected to use Railpack later")
	}

	result.Hints = append(result.Hints, scanReadinessHints(root)...)
	result.Hints = uniqueStrings(result.Hints)

	return result
}

func detectProjectType(root string) (string, []string) {
	switch {
	case fileExists(filepath.Join(root, "go.mod")):
		return ProjectTypeGo, []string{"Detected Go project via go.mod"}
	case fileExists(filepath.Join(root, "package.json")):
		return ProjectTypeNode, []string{"Detected Node project via package.json"}
	case fileExists(filepath.Join(root, "requirements.txt")):
		return ProjectTypePython, []string{"Detected Python project via requirements.txt"}
	case fileExists(filepath.Join(root, "pyproject.toml")):
		return ProjectTypePython, []string{"Detected Python project via pyproject.toml"}
	case fileExists(filepath.Join(root, "Cargo.toml")):
		return ProjectTypeRust, []string{"Detected Rust project via Cargo.toml"}
	default:
		return ProjectTypeUnknown, nil
	}
}

func scanReadinessHints(root string) []string {
	hints := make([]string, 0)
	scannedFiles := 0

	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || scannedFiles >= maxScannedFiles {
			return nil
		}
		if entry.IsDir() {
			if shouldSkipDir(entry.Name()) && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		if !isScannableFile(path) {
			return nil
		}

		info, err := entry.Info()
		if err != nil || info.Size() > maxScannedFileSize {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		scannedFiles++
		text := string(content)
		if strings.Contains(text, "localhost") || strings.Contains(text, "127.0.0.1") {
			hints = append(hints, "Found hardcoded localhost reference")
		}
		if hasHighDBPoolConfig(text) {
			hints = append(hints, "Suspicious DB pool config might be high")
		}

		return nil
	})

	return hints
}

func hasHighDBPoolConfig(content string) bool {
	pattern := regexp.MustCompile(`(?i)(max_open_conns|maxopenconns|maxidleconns|max_pool_size|maxpoolsize|pool_size|connectionlimit)\D{0,20}(\d+)`)
	matches := pattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		value, err := strconv.Atoi(match[2])
		if err == nil && value >= 100 {
			return true
		}
	}

	return false
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func shouldSkipDir(name string) bool {
	switch name {
	case ".git", "node_modules", "vendor", "target", "dist", "build", ".next", ".venv", "venv":
		return true
	default:
		return false
	}
}

func isScannableFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go", ".js", ".jsx", ".ts", ".tsx", ".json", ".py", ".rs", ".toml", ".yaml", ".yml", ".env", ".txt", ".md":
		return true
	default:
		return false
	}
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	unique := make([]string, 0, len(values))

	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}

		seen[value] = struct{}{}
		unique = append(unique, value)
	}

	return unique
}
