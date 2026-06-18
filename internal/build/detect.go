package build

import (
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

// DetectRuntime inspects the contents of repoPath and returns the
// detected runtime based on marker files, checked in priority order.
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

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}