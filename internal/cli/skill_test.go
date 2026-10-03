package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oscarhugopaz/earth-cli/internal/skill"
)

func TestSkillInstallCLI(t *testing.T) {
	project, home := t.TempDir(), t.TempDir()
	t.Chdir(project)
	// Installing a skill must not need working Earth config or credentials.
	if err := os.MkdirAll(filepath.Join(home, ".config", "earth"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".config", "earth", "config.yaml"), []byte("invalid: ["), 0o600); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) (int, string, string) {
		var out, stderr bytes.Buffer
		code := Run(context.Background(), args, Environment{
			Stdout: &out, Stderr: &stderr,
			LookupEnv: func(key string) (string, bool) { return home, key == "HOME" },
		})
		return code, out.String(), stderr.String()
	}
	code, out, stderr := run("skill", "install", "--json")
	if code != 0 || stderr != "" {
		t.Fatalf("exit=%d stderr=%s", code, stderr)
	}
	var results []skill.Result
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Path != filepath.Join(project, ".agents", "skills", "earth", "SKILL.md") {
		t.Fatalf("results=%v", results)
	}
	code, out, stderr = run("skill", "install")
	if code != 0 || !strings.Contains(out, "unchanged:") {
		t.Fatalf("exit=%d out=%s stderr=%s", code, out, stderr)
	}
	code, out, stderr = run("skill", "install", "--global", "--all", "--json")
	if code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, stderr)
	}
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Fatalf("all results=%v", results)
	}
	for _, result := range results {
		if !strings.HasPrefix(result.Path, home+string(filepath.Separator)) {
			t.Fatalf("wrong scope: %v", result)
		}
	}
	path := filepath.Join(project, ".agents", "skills", "earth", "SKILL.md")
	if err := os.WriteFile(path, []byte("custom"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, stderr = run("skill", "install")
	if code != 1 || out != "" || !strings.Contains(stderr, "--force") {
		t.Fatalf("exit=%d out=%s stderr=%s", code, out, stderr)
	}
	code, out, stderr = run("skill", "install", "--force")
	if code != 0 || !strings.Contains(out, "updated:") {
		t.Fatalf("exit=%d out=%s stderr=%s", code, out, stderr)
	}
	for _, args := range [][]string{
		{"skill", "install", "--agent", "unknown"},
		{"skill", "install", "--agent", "claude", "--all"},
		{"skill", "install", "extra"},
	} {
		code, _, stderr = run(args...)
		if code != 2 {
			t.Fatalf("args=%v exit=%d stderr=%s", args, code, stderr)
		}
	}
}
