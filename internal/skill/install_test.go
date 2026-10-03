package skill

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func noEnv(string) (string, bool) { return "", false }

func TestTargets(t *testing.T) {
	project, home := t.TempDir(), t.TempDir()
	cases := []struct {
		agent         string
		local, global string
	}{
		{"", ".agents/skills/earth", ".agents/skills/earth"},
		{"agents", ".agents/skills/earth", ".agents/skills/earth"},
		{"codex", ".agents/skills/earth", ".agents/skills/earth"},
		{"claude", ".claude/skills/earth", ".claude/skills/earth"},
		{"opencode", ".opencode/skills/earth", ".config/opencode/skills/earth"},
		{"pi", ".pi/skills/earth", ".pi/agent/skills/earth"},
		{"hermes", ".hermes/skills/earth", ".hermes/skills/earth"},
	}
	for _, tc := range cases {
		t.Run(tc.agent, func(t *testing.T) {
			for _, global := range []bool{false, true} {
				base, relative := project, tc.local
				if global {
					base, relative = home, tc.global
				}
				paths, err := Targets(project, home, tc.agent, global, false, noEnv)
				if err != nil {
					t.Fatal(err)
				}
				if len(paths) != 1 || paths[0] != filepath.Join(base, filepath.FromSlash(relative)) {
					t.Fatalf("global=%v paths=%v", global, paths)
				}
			}
		})
	}
	for _, global := range []bool{false, true} {
		paths, err := Targets(project, home, "", global, true, noEnv)
		if err != nil {
			t.Fatal(err)
		}
		want := 2
		if global {
			want = 3
		}
		if len(paths) != want {
			t.Fatalf("all global=%v paths=%v", global, paths)
		}
	}
	if _, err := Targets(project, home, "other", false, false, noEnv); err == nil {
		t.Fatal("unknown agent accepted")
	}
	if _, err := Targets(project, home, "claude", false, true, noEnv); err == nil {
		t.Fatal("agent with all accepted")
	}
	if _, err := Targets(project, "", "", true, false, noEnv); err == nil {
		t.Fatal("missing home accepted")
	}
}

func TestGlobalOverridesAndDeduplication(t *testing.T) {
	project, home, custom := t.TempDir(), t.TempDir(), t.TempDir()
	lookup := func(key string) (string, bool) {
		if key == "PI_CODING_AGENT_DIR" || key == "HERMES_HOME" {
			return custom, true
		}
		return "", false
	}
	for _, agent := range []string{"pi", "hermes"} {
		paths, err := Targets(project, home, agent, true, false, lookup)
		if err != nil || len(paths) != 1 || paths[0] != filepath.Join(custom, "skills", "earth") {
			t.Fatalf("agent=%s paths=%v err=%v", agent, paths, err)
		}
		paths, err = Targets(project, home, agent, false, false, lookup)
		if err != nil || strings.HasPrefix(paths[0], custom) {
			t.Fatalf("global override affected local paths: %v, %v", paths, err)
		}
	}
	paths, err := Targets(project, home, "", true, true, func(key string) (string, bool) {
		return filepath.Join(home, ".agents"), key == "HERMES_HOME"
	})
	if err != nil || len(paths) != 2 {
		t.Fatalf("dedup paths=%v err=%v", paths, err)
	}
}

func TestInstallAndUpdate(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "skills", "earth")
	results, err := Install([]string{dir}, false)
	if err != nil || len(results) != 1 || results[0].Status != "installed" {
		t.Fatalf("results=%v err=%v", results, err)
	}
	path := filepath.Join(dir, "SKILL.md")
	data, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(data, content) {
		t.Fatalf("embedded content mismatch: %v", err)
	}
	before, _ := os.Stat(path)
	results, err = Install([]string{dir}, false)
	if err != nil || results[0].Status != "unchanged" {
		t.Fatalf("results=%v err=%v", results, err)
	}
	after, _ := os.Stat(path)
	if !os.SameFile(before, after) || !before.ModTime().Equal(after.ModTime()) {
		t.Fatal("identical file was rewritten")
	}
	if err := os.WriteFile(path, []byte("user edits"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Install([]string{dir}, false); err == nil {
		t.Fatal("different file overwritten without force")
	}
	data, _ = os.ReadFile(path)
	if string(data) != "user edits" {
		t.Fatal("user edits lost")
	}
	extra := filepath.Join(dir, "notes.md")
	if err := os.WriteFile(extra, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	results, err = Install([]string{dir}, true)
	if err != nil || results[0].Status != "updated" {
		t.Fatalf("results=%v err=%v", results, err)
	}
	data, _ = os.ReadFile(extra)
	if string(data) != "keep" {
		t.Fatal("extra file changed")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 2 {
		t.Fatalf("temporary files remain: %v", entries)
	}
}

func TestInstallPreflightsAllTargets(t *testing.T) {
	base := t.TempDir()
	first, second := filepath.Join(base, "first"), filepath.Join(base, "second")
	if err := os.Mkdir(second, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(second, "SKILL.md"), []byte("different"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Install([]string{first, second}, false); err == nil {
		t.Fatal("conflict accepted")
	}
	if _, err := os.Stat(first); !os.IsNotExist(err) {
		t.Fatal("first destination modified before conflict was detected")
	}
}

func TestInstallRejectsSymlinks(t *testing.T) {
	base, outside := t.TempDir(), t.TempDir()
	file := filepath.Join(outside, "keep.md")
	if err := os.WriteFile(file, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(base, "linked")); err != nil {
		t.Skip(err)
	}
	if _, err := Install([]string{filepath.Join(base, "linked", "earth")}, true); err == nil {
		t.Fatal("directory symlink accepted")
	}
	dir := filepath.Join(base, "earth")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(file, filepath.Join(dir, "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := Install([]string{dir}, true); err == nil {
		t.Fatal("file symlink accepted")
	}
	data, _ := os.ReadFile(file)
	if string(data) != "keep" {
		t.Fatal("symlink target modified")
	}
}

func TestBundledSkill(t *testing.T) {
	parts := strings.SplitN(string(content), "---", 3)
	if len(parts) != 3 {
		t.Fatal("missing skill frontmatter")
	}
	var front struct {
		Name        string
		Description string
	}
	if err := yaml.Unmarshal([]byte(parts[1]), &front); err != nil {
		t.Fatal(err)
	}
	if front.Name != "earth" || !strings.Contains(front.Description, "Use when") || len(front.Description) > 1024 {
		t.Fatalf("invalid metadata: %+v", front)
	}
	if lines := strings.Count(string(content), "\n"); lines >= 100 {
		t.Fatalf("skill too long: %d lines", lines)
	}
}
