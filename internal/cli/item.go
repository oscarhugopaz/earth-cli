package cli

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/oscarhugopaz/earth-cli/internal/apperr"
	"github.com/oscarhugopaz/earth-cli/internal/output"
	"github.com/oscarhugopaz/earth-cli/internal/provider"
)

func newItemCommand(env Environment) *cobra.Command {
	var (
		downloadAsset string
		downloadDir   string
	)

	cmd := &cobra.Command{
		Use:   "item <collection> <item-id>",
		Short: "Show a single item and its assets",
		Long: `Show a single STAC item, including its assets and their URLs.

  earth item sentinel-2-l2a S2B_MSIL2A_20260914T143739_N0512_R096_T19HCC_20260914T195453

Download one asset (HTTP/HTTPS assets only):

  earth item sentinel-2-l2a <item-id> --download thumbnail
  earth item sentinel-2-l2a <item-id> --download thumbnail --download-dir ./out`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			session, err := newSession(cmd, env)
			if err != nil {
				return err
			}
			p, err := session.selectedProvider()
			if err != nil {
				return err
			}

			item, err := p.Item(cmd.Context(), args[0], args[1])
			if err != nil {
				return err
			}

			if downloadAsset != "" {
				return downloadItemAsset(cmd, session.printer, item, downloadAsset, downloadDir)
			}

			if session.json {
				return session.printer.JSONValue(item)
			}
			printItem(session.printer, item)
			return nil
		},
	}

	cmd.Flags().StringVar(&downloadAsset, "download", "", "download the named asset (HTTP/HTTPS assets only)")
	cmd.Flags().StringVar(&downloadDir, "download-dir", ".", "directory to write downloaded assets into")
	return cmd
}

// downloadItemAsset fetches one asset to disk. Only HTTP/HTTPS assets are
// supported; other schemes (for example Sentinel's s3://eodata) get an
// actionable error rather than a silent failure.
func downloadItemAsset(cmd *cobra.Command, printer *output.Printer, item provider.Observation, name, dir string) error {
	var asset *provider.Asset
	for i := range item.AssetDetails {
		if item.AssetDetails[i].Name == name {
			asset = &item.AssetDetails[i]
			break
		}
	}
	if asset == nil {
		return apperr.Usage("item %q has no asset %q\n\nAvailable assets:\n  %s",
			item.ID, name, strings.Join(item.Assets, "\n  "))
	}
	if asset.Href == "" {
		return apperr.New("asset %q has no download URL", name)
	}

	scheme := strings.ToLower(strings.SplitN(asset.Href, ":", 2)[0])
	if scheme != "http" && scheme != "https" {
		return apperr.New("asset %q uses the %q scheme, which earth cannot download directly.\n"+
			"Use the object storage credentials for that provider, or pick an HTTP asset (for example \"thumbnail\").",
			name, scheme)
	}

	target := filepath.Join(dir, assetFileName(asset))

	req, err := http.NewRequestWithContext(cmd.Context(), http.MethodGet, asset.Href, nil)
	if err != nil {
		return apperr.Wrap(err, "build download request for %q", name)
	}
	req.Header.Set("User-Agent", "earth-cli")

	timeout, _ := cmd.Flags().GetDuration("timeout")
	resp, err := (&http.Client{Timeout: timeout}).Do(req)
	if err != nil {
		return apperr.Wrap(err, "download asset %q", name)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return apperr.New("download failed for asset %q: HTTP %s", name, resp.Status)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return apperr.Wrap(err, "create download directory %q", dir)
	}

	file, err := os.CreateTemp(dir, ".earth-download-*")
	if err != nil {
		return apperr.Wrap(err, "create %q", target)
	}
	defer os.Remove(file.Name())
	written, copyErr := io.Copy(file, resp.Body)
	closeErr := file.Close()
	if copyErr != nil {
		return apperr.Wrap(copyErr, "write %q", target)
	}
	if closeErr != nil {
		return apperr.Wrap(closeErr, "close %q", target)
	}
	// A hard link publishes the complete download without overwriting files.
	if err := os.Link(file.Name(), target); err != nil {
		return apperr.Wrap(err, "save %q (existing files are not overwritten)", target)
	}
	jsonOut, _ := cmd.Flags().GetBool("json")
	if jsonOut {
		return printer.JSONValue(struct {
			Asset string `json:"asset"`
			Path  string `json:"path"`
			Bytes int64  `json:"bytes"`
		}{asset.Name, target, written})
	}

	printer.Line("Downloaded %s (%s) to %s", asset.Name, humanBytes(written), target)
	return nil
}

// assetFileName picks a safe file name for an asset, preferring the asset name
// and keeping a sensible extension from the URL when one is present.
func assetFileName(asset *provider.Asset) string {
	base := sanitizeName(asset.Name)
	if base == "" {
		base = "asset"
	}

	if u, err := url.Parse(asset.Href); err == nil {
		if ext := filepath.Ext(u.Path); ext != "" && len(ext) <= 6 && !strings.ContainsAny(ext, "/\\") {
			return base + ext
		}
	}
	switch {
	case strings.Contains(asset.Type, "tiff"):
		return base + ".tif"
	case strings.Contains(asset.Type, "png"):
		return base + ".png"
	case strings.Contains(asset.Type, "jpeg"), strings.Contains(asset.Type, "jpg"):
		return base + ".jpg"
	case strings.Contains(asset.Type, "json"):
		return base + ".json"
	case strings.Contains(asset.Type, "jp2"):
		return base + ".jp2"
	default:
		return base
	}
}

// sanitizeName keeps asset names safe for filesystem use.
func sanitizeName(name string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", " ", "_")
	return strings.Trim(replacer.Replace(name), "._")
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for value := n / unit; value >= unit; value /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}

func printItem(printer *output.Printer, item provider.Observation) {
	printer.Field("ID", item.ID)
	if item.Collection != "" {
		printer.Field("Collection", item.Collection)
	}
	if item.DateTime != nil {
		printer.Field("Date", item.DateTime.UTC().Format("2006-01-02T15:04:05Z"))
	}
	if item.CloudCover != nil {
		printer.Field("Cloud cover", fmt.Sprintf("%.1f%%", *item.CloudCover))
	}
	if item.ItemURL != "" {
		printer.Field("Item URL", item.ItemURL)
	}

	if len(item.AssetDetails) == 0 {
		return
	}
	printer.Line("")
	printer.Line("Assets (%d)", len(item.AssetDetails))
	rows := make([][]string, 0, len(item.AssetDetails))
	for _, asset := range item.AssetDetails {
		rows = append(rows, []string{asset.Name, asset.Type, asset.Href})
	}
	printer.Table([]string{"NAME", "TYPE", "HREF"}, rows)
}
