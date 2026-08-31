package build

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Runtime string

const (
	RuntimeNode   Runtime = "node"
	RuntimePython Runtime = "python"
	RuntimeGo     Runtime = "go"
	RuntimeStatic Runtime = "static"
)

var ErrUnknownRuntime = fmt.Errorf("could not detect a supported runtime in repository")

func DetectRuntime(repoPath string) (Runtime, error) {
	checks := []struct {
		file    string
		runtime Runtime
	}{
		{"package.json", RuntimeNode},
		{"go.mod", RuntimeGo},
		{"requirements.txt", RuntimePython},
		{"pyproject.toml", RuntimePython},
		{"index.html", RuntimeStatic},
	}

	for _, check := range checks {
		path := filepath.Join(repoPath, check.file)
		if fileExists(path) {
			return check.runtime, nil
		}
	}

	return "", ErrUnknownRuntime
}

func ValidateRunnable(repoPath string, runtime Runtime) error {
	switch runtime {
	case RuntimeNode:
		return validateNode(repoPath)
	case RuntimePython:
		return validatePython(repoPath)
	case RuntimeGo:
		return validateGo(repoPath)
	case RuntimeStatic:
		return nil
	default:
		return fmt.Errorf("unknown runtime: %s", runtime)
	}
}

func validateNode(repoPath string) error {
	pkgPath := filepath.Join(repoPath, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return fmt.Errorf("could not read package.json: %w", err)
	}

	var pkg struct {
		Scripts map[string]string `json:"scripts"`
		Main    string            `json:"main"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return fmt.Errorf("could not parse package.json: %w", err)
	}

	if _, ok := pkg.Scripts["start"]; ok {
		return nil
	}

	if pkg.Main != "" {
		if fileExists(filepath.Join(repoPath, pkg.Main)) {
			return nil
		}
	}

	return fmt.Errorf(`no "start" script found in package.json — add: "scripts": { "start": "node index.js" }`)
}

func validatePython(repoPath string) error {
	entryPoints := []string{"app.py", "main.py", "wsgi.py", "manage.py", "server.py"}
	for _, entry := range entryPoints {
		if fileExists(filepath.Join(repoPath, entry)) {
			return nil
		}
	}
	return fmt.Errorf("no Python entry point found — add app.py, main.py, or wsgi.py")
}

func validateGo(repoPath string) error {
	if fileExists(filepath.Join(repoPath, "main.go")) {
		return nil
	}
	if dirExists(filepath.Join(repoPath, "cmd")) {
		return nil
	}
	return fmt.Errorf("no main.go or cmd/ directory found — Go apps need a main package")
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}