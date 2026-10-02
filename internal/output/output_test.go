package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestTable(t *testing.T) {
	var out bytes.Buffer
	printer := New(&out, &out, false, false)
	printer.Table(
		[]string{"NAME", "TYPE", "STATUS"},
		[][]string{
			{"copernicus", "stac", "available"},
		},
	)

	want := "NAME        TYPE  STATUS\n" +
		"copernicus  stac  available\n"
	if out.String() != want {
		t.Fatalf("table output:\n%q\nwant:\n%q", out.String(), want)
	}
}

func TestJSONValue(t *testing.T) {
	var out bytes.Buffer
	printer := New(&out, &out, true, false)
	if err := printer.JSONValue(map[string]string{"name": "sentinel-2-l2a <a>"}); err != nil {
		t.Fatalf("JSONValue returned error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "<a>") {
		t.Fatalf("JSON should not HTML-escape: %q", got)
	}
	if !strings.Contains(got, "\n  \"name\"") {
		t.Fatalf("JSON should be indented: %q", got)
	}
}

func TestField(t *testing.T) {
	var out bytes.Buffer
	printer := New(&out, &out, false, false)
	printer.Field("Observation", "vegetation")
	if got := out.String(); got != "Observation   vegetation\n" {
		t.Fatalf("Field = %q", got)
	}
}

func TestSupportsColor(t *testing.T) {
	var buffer bytes.Buffer

	if SupportsColor(&buffer, false, func(string) (string, bool) { return "", false }) {
		t.Fatal("buffer is not a terminal, color should be disabled")
	}
	if SupportsColor(&buffer, true, func(string) (string, bool) { return "", false }) {
		t.Fatal("noColor should disable color")
	}
	noColorEnv := func(key string) (string, bool) {
		if key == "NO_COLOR" {
			return "1", true
		}
		return "", false
	}
	if SupportsColor(&buffer, false, noColorEnv) {
		t.Fatal("NO_COLOR should disable color")
	}
}
