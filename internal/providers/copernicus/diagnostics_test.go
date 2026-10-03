package copernicus

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDiagnosticURL(t *testing.T) {
	got := diagnosticURL("https://user:secret@example.com/search?token=private#fragment")
	if got != "https://example.com/search" {
		t.Fatalf("unsafe diagnostic URL: %s", got)
	}
}

func TestDiagnosticsDoNotLogSecrets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{}`)) }))
	defer server.Close()
	var log strings.Builder
	p := New(server.URL, WithDebug(func(format string, args ...any) { fmt.Fprintf(&log, format+"\n", args...) }))
	err := p.request(context.Background(), http.MethodPost, server.URL+"?token=querysecret", []byte(`{"client_secret":"bodysecret"}`), "bearersecret", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"querysecret", "bodysecret", "bearersecret"} {
		if strings.Contains(log.String(), secret) {
			t.Fatalf("secret leaked in diagnostics")
		}
	}
	if !strings.Contains(log.String(), "200 OK") {
		t.Fatalf("missing response status: %s", log.String())
	}
}
