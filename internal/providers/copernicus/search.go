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

// Search runs a STAC item search. In this release it performs a single
// request; the provider-neutral SearchRequest leaves room for pagination later.
func (p *Provider) Search(ctx context.Context, req provider.SearchRequest) ([]provider.Observation, error) {
	collection := strings.TrimSpace(req.Collection)
	if collection == "" {
		return nil, fmt.Errorf("a collection is required to search %s", displayName)
	}

	body := map[string]any{
		"collections": []string{collection},
	}
	if req.BBox != nil {
		body["bbox"] = req.BBox.Slice()
	}
	if interval := datetimeInterval(req.Start, req.End); interval != "" {
		body["datetime"] = interval
	}
	if req.Limit > 0 {
		body["limit"] = req.Limit
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encode %s search request: %w", displayName, err)
	}

	var response searchResponse
	if err := p.do(ctx, http.MethodPost, p.baseURL+"/search", payload, &response); err != nil {
		return nil, err
	}

	observations := make([]provider.Observation, 0, len(response.Features))
	for _, feature := range response.Features {
		observations = append(observations, normalizeObservation(feature, collection, p))
	}
	return observations, nil
}

func normalizeObservation(feature stacFeature, requestedCollection string, p *Provider) provider.Observation {
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

	for _, link := range feature.Links {
		if link.Rel == "self" || link.Rel == "item" {
			observation.ItemURL = link.Href
			break
		}
	}

	return observation
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
