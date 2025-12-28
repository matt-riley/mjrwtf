package ci

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func scriptPath(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "scripts", "workflow-policy-check.sh"))
}

func runPolicy(t *testing.T, root string) (string, error) {
	t.Helper()
	cmd := exec.Command("bash", scriptPath(t), "--root", root)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func TestWorkflowPolicyCheck_PassesWithPinnedActionsAndCodegen(t *testing.T) {
	root := t.TempDir()

	writeFile(t, filepath.Join(root, ".github/workflows/a.yml"), "name: A\nsteps:\n  - uses: actions/checkout@0123456789abcdef0123456789abcdef01234567\n")
	writeFile(t, filepath.Join(root, ".github/workflows/goreleaser.yml"), "name: GoReleaser\njobs:\n  x:\n    steps:\n      - name: Install sqlc\n        run: echo ok\n      - name: Install templ\n        run: echo ok\n")
	writeFile(t, filepath.Join(root, ".goreleaser.yaml"), "before:\n  hooks:\n    - go mod download\n    - sqlc generate\n    - templ generate\n")

	out, err := runPolicy(t, root)
	if err != nil {
		t.Fatalf("expected success, got error: %v\noutput:\n%s", err, out)
	}
}

func TestWorkflowPolicyCheck_FailsOnVersionTag(t *testing.T) {
	root := t.TempDir()

	writeFile(t, filepath.Join(root, ".github/workflows/a.yml"), "name: A\nsteps:\n  - uses: actions/checkout@v6.1.0\n")
	writeFile(t, filepath.Join(root, ".github/workflows/goreleaser.yml"), "name: GoReleaser\njobs:\n  x:\n    steps:\n      - name: Install sqlc\n        run: echo ok\n      - name: Install templ\n        run: echo ok\n")
	writeFile(t, filepath.Join(root, ".goreleaser.yaml"), "before:\n  hooks:\n    - go mod download\n    - sqlc generate\n    - templ generate\n")

	out, err := runPolicy(t, root)
	if err == nil {
		t.Fatalf("expected failure, got success\noutput:\n%s", out)
	}
}

func TestWorkflowPolicyCheck_FailsWhenCodegenMissing(t *testing.T) {
	root := t.TempDir()

	writeFile(t, filepath.Join(root, ".github/workflows/a.yml"), "name: A\nsteps:\n  - uses: actions/checkout@0123456789abcdef0123456789abcdef01234567\n")
	writeFile(t, filepath.Join(root, ".github/workflows/goreleaser.yml"), "name: GoReleaser\njobs:\n  x:\n    steps:\n      - name: Install sqlc\n        run: echo ok\n      - name: Install templ\n        run: echo ok\n")
	writeFile(t, filepath.Join(root, ".goreleaser.yaml"), "before:\n  hooks:\n    - go mod download\n")

	out, err := runPolicy(t, root)
	if err == nil {
		t.Fatalf("expected failure, got success\noutput:\n%s", out)
	}
}
