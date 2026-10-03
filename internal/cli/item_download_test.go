package cli

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oscarhugopaz/earth-cli/internal/output"
	"github.com/oscarhugopaz/earth-cli/internal/provider"
	"github.com/spf13/cobra"
)

func TestDownloadAsset(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("image")) }))
	defer server.Close()
	dir := t.TempDir()
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().Bool("json", true, "")
	var out bytes.Buffer
	printer := output.New(&out, &out, true, false)
	item := provider.Observation{AssetDetails: []provider.Asset{{Name: "thumb", Href: server.URL + "/image.jpg"}}}
	if err := downloadItemAsset(cmd, printer, item, "thumb", dir); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "thumb.jpg"))
	if err != nil || string(data) != "image" {
		t.Fatalf("data=%q err=%v", data, err)
	}
	if !strings.Contains(out.String(), `"bytes": 5`) {
		t.Fatal(out.String())
	}
	if err := downloadItemAsset(cmd, printer, item, "thumb", dir); err == nil {
		t.Fatal("overwrote existing file")
	}
	if err := downloadItemAsset(cmd, printer, item, "missing", dir); err == nil {
		t.Fatal("missing asset accepted")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("temporary files remain: %v", entries)
	}
}

func TestDownloadFailureDoesNotLeaveFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		w.Write([]byte("short"))
	}))
	defer server.Close()
	dir := t.TempDir()
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	var out bytes.Buffer
	printer := output.New(&out, &out, false, false)
	item := provider.Observation{AssetDetails: []provider.Asset{{Name: "thumb", Href: server.URL}}}
	if err := downloadItemAsset(cmd, printer, item, "thumb", dir); err == nil {
		t.Fatal("truncated download accepted")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Fatalf("partial files remain: %v", entries)
	}
	item.AssetDetails[0].Href = "s3://bucket/key"
	if err := downloadItemAsset(cmd, printer, item, "thumb", dir); err == nil {
		t.Fatal("s3 accepted")
	}
}
