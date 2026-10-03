// Package skill installs the portable Earth agent skill bundled with the CLI.
package skill

import (
	"bytes"
	"crypto/rand"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed SKILL.md
var content []byte

// Targets resolves skill directories without requiring agent executables.
// An empty agent selects the shared .agents location.
func Targets(project, home, agent string, global, all bool, lookup func(string) (string, bool)) ([]string, error) {
	if all && agent != "" {
		return nil, fmt.Errorf("--agent and --all are mutually exclusive")
	}
	base := project
	if global {
		base = home
		if base == "" {
			return nil, fmt.Errorf("cannot determine user home directory")
		}
	}
	shared := filepath.Join(base, ".agents", "skills", "earth")
	claude := filepath.Join(base, ".claude", "skills", "earth")
	hermes := filepath.Join(base, ".hermes", "skills", "earth")
	if global {
		if dir, ok := lookup("HERMES_HOME"); ok && dir != "" {
			hermes = filepath.Join(dir, "skills", "earth")
		}
	}
	var paths []string
	if all {
		paths = []string{shared, claude}
		if global {
			paths = append(paths, hermes)
		}
	} else {
		switch agent {
		case "", "agents", "codex":
			paths = []string{shared}
		case "claude":
			paths = []string{claude}
		case "hermes":
			paths = []string{hermes}
		case "opencode":
			dir := filepath.Join(base, ".opencode")
			if global {
				dir = filepath.Join(home, ".config", "opencode")
			}
			paths = []string{filepath.Join(dir, "skills", "earth")}
		case "pi":
			dir := filepath.Join(base, ".pi")
			if global {
				dir = filepath.Join(home, ".pi", "agent")
				if override, ok := lookup("PI_CODING_AGENT_DIR"); ok && override != "" {
					dir = override
				}
			}
			paths = []string{filepath.Join(dir, "skills", "earth")}
		default:
			return nil, fmt.Errorf("unknown agent %q; choose agents, claude, codex, opencode, pi, or hermes", agent)
		}
	}
	seen := map[string]bool{}
	var result []string
	for _, path := range paths {
		path, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		if !seen[path] {
			seen[path] = true
			result = append(result, path)
		}
	}
	return result, nil
}

// Result describes one installed SKILL.md: installed, updated, or unchanged.
type Result struct {
	Path   string `json:"path"`
	Status string `json:"status"`
}

// Install preflights all destinations before writing, then publishes each file
// atomically. Other files in the skill directory are never modified.
func Install(dirs []string, force bool) ([]Result, error) {
	results := make([]Result, 0, len(dirs))
	for _, dir := range dirs {
		if err := checkDirectories(dir, false); err != nil {
			return nil, err
		}
		path := filepath.Join(dir, "SKILL.md")
		status := "installed"
		info, err := os.Lstat(path)
		if err == nil {
			if !info.Mode().IsRegular() {
				return nil, fmt.Errorf("refusing non-regular skill file %q", path)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			if bytes.Equal(data, content) {
				status = "unchanged"
			} else if !force {
				return nil, fmt.Errorf("skill %q has different content; use --force to replace it", path)
			} else {
				status = "updated"
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		results = append(results, Result{Path: path, Status: status})
	}
	for i, result := range results {
		if result.Status == "unchanged" {
			continue
		}
		if err := writeSkill(filepath.Dir(result.Path), force); err != nil {
			return results[:i], fmt.Errorf("install %q: %w", result.Path, err)
		}
	}
	return results, nil
}

// checkDirectories rejects symlinks and non-directories rather than writing
// through them, including with --force.
func checkDirectories(dir string, create bool) error {
	parent := filepath.Dir(dir)
	if parent != dir {
		if err := checkDirectories(parent, create); err != nil {
			return err
		}
	}
	info, err := os.Lstat(dir)
	if errors.Is(err, os.ErrNotExist) {
		if !create {
			return nil
		}
		if err := os.Mkdir(dir, 0o755); err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}
		info, err = os.Lstat(dir)
	}
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing symlink or non-directory %q", dir)
	}
	return nil
}

func writeSkill(dir string, force bool) error {
	if err := checkDirectories(dir, true); err != nil {
		return err
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer root.Close()
	tmp := ".earth-skill-" + rand.Text()
	file, err := root.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	defer root.Remove(tmp)
	_, writeErr := file.Write(content)
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	if force {
		return root.Rename(tmp, "SKILL.md")
	}
	// Link is exclusive: a file created after preflight is not overwritten.
	return root.Link(tmp, "SKILL.md")
}
