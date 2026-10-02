package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func runCLI(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), args, Environment{
		Stdin:     bytes.NewReader(nil),
		Stdout:    &stdout,
		Stderr:    &stderr,
		LookupEnv: func(string) (string, bool) { return "", false },
	})
	return code, stdout.String(), stderr.String()
}

func TestVersionCommand(t *testing.T) {
	code, stdout, _ := runCLI(t, "version")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if stdout != "earth version dev\n" {
		t.Fatalf("stdout = %q", stdout)
	}
}

func TestVersionCommandJSON(t *testing.T) {
	code, stdout, _ := runCLI(t, "version", "--json")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	for _, fragment := range []string{`"version"`, `"commit"`, `"date"`} {
		if !strings.Contains(stdout, fragment) {
			t.Fatalf("stdout = %q, missing %q", stdout, fragment)
		}
	}
}

func TestProvidersCommand(t *testing.T) {
	code, stdout, _ := runCLI(t, "providers")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "copernicus") || !strings.Contains(stdout, "stac") || !strings.Contains(stdout, "available") {
		t.Fatalf("stdout = %q", stdout)
	}
}

func TestProvidersCommandJSON(t *testing.T) {
	code, stdout, _ := runCLI(t, "providers", "--json")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, `"name": "copernicus"`) {
		t.Fatalf("stdout = %q", stdout)
	}
}

func TestHelp(t *testing.T) {
	code, stdout, _ := runCLI(t, "--help")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	for _, command := range []string{"search", "collections", "observe", "providers", "version", "completion"} {
		if !strings.Contains(stdout, command) {
			t.Fatalf("help missing %q:\n%s", command, stdout)
		}
	}
}

func TestUsageErrors(t *testing.T) {
	cases := []struct {
		name   string
		args   []string
		stderr string
	}{
		{"search without collection", []string{"search", "--bbox", "-70.8,-33.6,-70.4,-33.3"}, "--collection is required"},
		{"search invalid bbox", []string{"search", "--collection", "sentinel-2-l2a", "--bbox", "bad"}, "invalid bbox"},
		{"search both area and bbox", []string{"search", "--collection", "sentinel-2-l2a", "--bbox", "-70.8,-33.6,-70.4,-33.3", "--area", "x.geojson"}, "mutually exclusive"},
		{"search since and from", []string{"search", "--collection", "sentinel-2-l2a", "--since", "30d", "--from", "2026-09-01"}, "--since cannot be combined with --from"},
		{"observe missing name", []string{"observe"}, "Available observations"},
		{"observe unknown", []string{"observe", "definitely-not-an-observation", "--bbox", "-70.8,-33.6,-70.4,-33.3"}, `unknown observation "definitely-not-an-observation"`},
		{"observe without area", []string{"observe", "vegetation"}, "requires an area of interest"},
		{"unknown command", []string{"bogus"}, "unknown command"},
		{"unknown flag", []string{"providers", "--nope"}, "unknown flag"},
		{"collection without id", []string{"collection"}, "accepts 1 arg"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, _, stderr := runCLI(t, tc.args...)
			if code != 2 {
				t.Fatalf("exit = %d, want 2 (stderr: %s)", code, stderr)
			}
			if !strings.Contains(stderr, tc.stderr) {
				t.Fatalf("stderr = %q, want %q", stderr, tc.stderr)
			}
		})
	}
}

func TestVersionFlag(t *testing.T) {
	code, stdout, _ := runCLI(t, "--version")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if stdout != "earth version dev\n" {
		t.Fatalf("stdout = %q", stdout)
	}
}
