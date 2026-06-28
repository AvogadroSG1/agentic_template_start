package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPythonFastAPIPyprojectIncludesHTTPX2ForTestClient(t *testing.T) {
	repoRoot := repoRoot(t)
	pyprojectPath := filepath.Join(repoRoot, "templates", "golden", "python-fastapi", "pyproject.toml.tmpl")

	contentBytes, err := os.ReadFile(pyprojectPath)
	if err != nil {
		t.Fatalf("read %s: %v", pyprojectPath, err)
	}

	content := string(contentBytes)
	if !strings.Contains(content, "\"httpx2") {
		t.Fatalf("%s missing httpx2 test dependency\n%s", pyprojectPath, content)
	}
}
