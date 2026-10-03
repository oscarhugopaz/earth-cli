package copernicus

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/oscarhugopaz/earth-cli/internal/provider"
)

const (
	// searchPageSize keeps each STAC request within the limits most catalogs
	// accept. CDSE, for example, caps heavy collections well below larger
	// values.
	searchPageSize = 100
	// maxSearchPages bounds pagination so a huge --limit cannot run away.
	maxSearchPages = 100
)

// Search runs a STAC item search, following pagination links until the
// requested number of items is reached.
func (p *Provider) Search(ctx context.Context, req provider.SearchRequest) ([]provider.Observation, error) {
	collection := strings.TrimSpace(req.Collection)
	if collection == "" {
		return nil, fmt.Errorf("a collection is required to search %s", p.name)
	}

	limit := req.Limit
	paginate := limit > 0

	pageSize := limit
	if !paginate || pageSize > searchPageSize {
		pageSize = searchPageSize
	}
	if !paginate {
		pageSize = 0
	}

	payload, err := json.Marshal(searchBody(req, pageSize))
	if err != nil {
		return nil, fmt.Errorf("encode %s search request: %w", p.name, err)
	}

	capacity := pageSize
	if capacity <= 0 {
		capacity = searchPageSize
	}
	observations := make([]provider.Observation, 0, capacity)

	var next *searchPage
	for page := 0; page < maxSearchPages; page++ {
		var response searchResponse
		switch {
		case next == nil:
			if err := p.do(ctx, http.MethodPost, p.baseURL+"/search", payload, &response); err != nil {
				return nil, err
			}
		case next.method == http.MethodGet:
			if err := p.do(ctx, http.MethodGet, next.href, nil, &response); err != nil {
				return nil, err
			}
		default:
			if err := p.do(ctx, next.method, next.href, next.body, &response); err != nil {
				return nil, err
			}
		}

		for _, feature := range response.Features {
			observations = append(observations, normalizeObservation(feature, collection, p, false))
			if paginate && len(observations) >= limit {
				return observations, nil
			}
		}

		if !paginate || len(response.Features) == 0 {
			break
		}
		next = nextSearchPage(response.Links, p.baseURL)
		if next == nil {
			break
		}
	}

	return observations, nil
}

func searchBody(req provider.SearchRequest, limit int) map[string]any {
	body := map[string]any{
		"collections": []string{strings.TrimSpace(req.Collection)},
	}
	if req.BBox != nil {
		body["bbox"] = req.BBox.Slice()
	}
	if interval := datetimeInterval(req.Start, req.End); interval != "" {
		body["datetime"] = interval
	}
	if limit > 0 {
		body["limit"] = limit
	}
	return body
}

// searchPage is a resolved next-page request.
type searchPage struct {
	method string
	href   string
	body   []byte
}

// nextSearchPage returns the next page, but only when it stays within the
// configured provider host.
func nextSearchPage(links []stacLink, baseURL string) *searchPage {
	for _, link := range links {
		if link.Rel != "next" || strings.TrimSpace(link.Href) == "" {
			continue
		}
		if !strings.HasPrefix(link.Href, baseURL) {
			return nil
		}

		method := strings.ToUpper(strings.TrimSpace(link.Method))
		if method == "" || method == http.MethodGet {
			return &searchPage{method: http.MethodGet, href: link.Href}
		}
		var body []byte
		if len(link.Body) > 0 {
			body = append([]byte(nil), link.Body...)
		}
		return &searchPage{method: method, href: link.Href, body: body}
	}
	return nil
}

func normalizeObservation(feature stacFeature, requestedCollection string, p *Provider, withAssetDetails bool) provider.Observation {
	observation := provider.Observation{
		ID:         feature.ID,
		Collection: feature.Collection,
		Provider:   p.Name(),
		BBox:       feature.BBox,
		Geometry:   feature.Geometry,
		Properties: feature.Properties,
	}
	if observation.Collection == "" {
		observation.Collection = requestedCollection
	}

	if raw, ok := feature.Properties["datetime"].(string); ok && raw != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, raw); err == nil {
			observation.DateTime = &parsed
		}
	}
	observation.CloudCover = numericProperty(feature.Properties, "eo:cloud_cover")
	observation.Assets = sortedKeys(feature.Assets)
	if withAssetDetails {
		observation.AssetDetails = assetDetails(feature.Assets)
	}

	for _, link := range feature.Links {
		if link.Rel == "self" || link.Rel == "item" {
			observation.ItemURL = link.Href
			break
		}
	}

	return observation
}

func assetDetails(assets map[string]stacAsset) []provider.Asset {
	names := sortedKeys(assets)
	details := make([]provider.Asset, 0, len(names))
	for _, name := range names {
		asset := assets[name]
		details = append(details, provider.Asset{
			Name:  name,
			Href:  asset.Href,
			Type:  asset.Type,
			Title: asset.Title,
			Roles: asset.Roles,
		})
	}
	return details
}

func numericProperty(properties map[string]any, key string) *float64 {
	value, ok := properties[key]
	if !ok {
		return nil
	}
	switch typed := value.(type) {
	case float64:
		return &typed
	case int:
		converted := float64(typed)
		return &converted
	case json.Number:
		if converted, err := typed.Float64(); err == nil {
			return &converted
		}
	case string:
		if converted, err := strconv.ParseFloat(strings.TrimSpace(typed), 64); err == nil {
			return &converted
		}
	}
	return nil
}

func datetimeInterval(start, end *time.Time) string {
	switch {
	case start != nil && end != nil:
		return start.UTC().Format(time.RFC3339) + "/" + end.UTC().Format(time.RFC3339)
	case start != nil:
		return start.UTC().Format(time.RFC3339) + "/.."
	case end != nil:
		return "../" + end.UTC().Format(time.RFC3339)
	default:
		return ""
	}
}
