package cli

import (
	"strings"
	"testing"
)

func TestIndicesCommand(t *testing.T) {
	code, stdout, stderr := runCLI(t, "indices")
	if code != 0 {
		t.Fatalf("exit = %d (stderr: %s)", code, stderr)
	}
	for _, name := range []string{"ndvi", "evi", "savi", "ndre", "ndmi", "ndwi", "mndwi", "ndbi", "nbr"} {
		if !strings.Contains(stdout, name) {
			t.Fatalf("indices output missing %q:\n%s", name, stdout)
		}
	}
}

func TestIndicesCommandJSON(t *testing.T) {
	code, stdout, _ := runCLI(t, "indices", "--json")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, `"name": "ndvi"`) {
		t.Fatalf("stdout = %q", stdout)
	}
	if !strings.Contains(stdout, `"formula"`) {
		t.Fatalf("stdout = %q", stdout)
	}
}

func TestObserveListsNewObservations(t *testing.T) {
	code, _, stderr := runCLI(t, "observe")
	if code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
	for _, name := range []string{"vegetation", "flood", "burnt-area", "moisture"} {
		if !strings.Contains(stderr, name) {
			t.Fatalf("stderr missing %q:\n%s", name, stderr)
		}
	}
}
