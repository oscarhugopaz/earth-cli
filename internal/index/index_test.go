package index

import (
	"strings"
	"testing"
)

func TestDefinitionsValid(t *testing.T) {
	names := Names()
	if len(names) < 9 {
		t.Fatalf("expected at least 9 indices, got %d: %v", len(names), names)
	}

	for _, def := range All() {
		t.Run(def.Name, func(t *testing.T) {
			if def.Title == "" || def.Description == "" || def.Formula == "" {
				t.Fatalf("incomplete definition: %+v", def)
			}
			if len(def.Bands) == 0 {
				t.Fatal("index declares no bands")
			}
			if !strings.HasPrefix(def.Evalscript, "//VERSION=3") {
				t.Fatalf("evalscript missing version header")
			}
			// Every band must be an input and the output id must be produced.
			for _, band := range def.Bands {
				if !strings.Contains(def.Evalscript, `"`+band+`"`) {
					t.Fatalf("evalscript does not request band %s", band)
				}
			}
			if !strings.Contains(def.Evalscript, `id: "`+def.OutputID+`"`) {
				t.Fatalf("evalscript does not declare output %s", def.OutputID)
			}
			// Cloud/shadow/snow masking must be present in every index.
			if !strings.Contains(def.Evalscript, "SCL") || !strings.Contains(def.Evalscript, "dataMask") {
				t.Fatal("evalscript must mask clouds/shadows/snow")
			}
		})
	}
}

func TestLookup(t *testing.T) {
	if _, ok := Lookup("NDVI"); !ok {
		t.Fatal("lookup should be case-insensitive")
	}
	if _, ok := Lookup(" ndwi "); !ok {
		t.Fatal("lookup should trim whitespace")
	}
	if _, ok := Lookup("nope"); ok {
		t.Fatal("unknown index should not resolve")
	}
}

func TestUnknownErrorMentionsAvailable(t *testing.T) {
	err := UnknownError("foo")
	for _, fragment := range []string{`"foo"`, "Available indices", "ndvi", "nbr", "evi"} {
		if !strings.Contains(err.Error(), fragment) {
			t.Fatalf("error %q missing %q", err.Error(), fragment)
		}
	}
}

func TestNamesAreSorted(t *testing.T) {
	names := Names()
	for i := 1; i < len(names); i++ {
		if names[i-1] > names[i] {
			t.Fatalf("names not sorted: %v", names)
		}
	}
}
