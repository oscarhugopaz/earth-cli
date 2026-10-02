package copernicus

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/oscarhugopaz/earth-cli/internal/provider"
)

const (
	collectionsPageSize = 1000
	collectionsHardCap  = 5000
)

// Collections lists collections, following STAC pagination links. A limit of
// zero (or less) returns every collection the catalog exposes, up to an
// internal safety cap.
func (p *Provider) Collections(ctx context.Context, limit int) ([]provider.Collection, error) {
	out := make([]provider.Collection, 0, 64)
	next := ""

	for {
		requestURL, err := p.collectionsURL(next)
		if err != nil {
			return nil, err
		}
		if requestURL == "" {
			break
		}

		var page collectionsResponse
		if err := p.do(ctx, http.MethodGet, requestURL, nil, &page); err != nil {
			return nil, err
		}

		for _, collection := range page.Collections {
			out = append(out, normalizeCollection(collection, p))
			if limit > 0 && len(out) >= limit {
				return out, nil
			}
		}

		next = nextLink(page.Links, p.baseURL)
		if next == "" || len(out) >= collectionsHardCap {
			break
		}
	}

	return out, nil
}

func (p *Provider) collectionsURL(next string) (string, error) {
	if next != "" {
		parsed, err := url.Parse(next)
		if err != nil {
			return "", fmt.Errorf("invalid %s pagination link %q: %w", displayName, next, err)
		}
		return parsed.String(), nil
	}

	endpoint, err := url.Parse(p.baseURL + "/collections")
	if err != nil {
		return "", fmt.Errorf("invalid %s STAC endpoint %q: %w", displayName, p.baseURL, err)
	}
	query := endpoint.Query()
	query.Set("limit", strconv.Itoa(collectionsPageSize))
	endpoint.RawQuery = query.Encode()
	return endpoint.String(), nil
}

// nextLink returns the absolute next-page URL, but only when it stays within
// the configured provider host.
func nextLink(links []stacLink, baseURL string) string {
	for _, link := range links {
		if link.Rel != "next" || strings.TrimSpace(link.Href) == "" {
			continue
		}
		if !strings.HasPrefix(link.Href, baseURL) {
			return ""
		}
		return link.Href
	}
	return ""
}

// Collection fetches and normalizes a single collection.
func (p *Provider) Collection(ctx context.Context, id string) (provider.Collection, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return provider.Collection{}, fmt.Errorf("collection id is required")
	}

	endpoint := p.baseURL + "/collections/" + encodePathSegment(id)
	var collection stacCollection
	if err := p.do(ctx, http.MethodGet, endpoint, nil, &collection); err != nil {
		var responseErr *provider.ResponseError
		if errors.As(err, &responseErr) && responseErr.Code == http.StatusNotFound {
			return provider.Collection{}, fmt.Errorf("collection %q was not found in the %s STAC catalog", id, displayName)
		}
		return provider.Collection{}, err
	}
	return normalizeCollection(collection, p), nil
}

func normalizeCollection(collection stacCollection, p *Provider) provider.Collection {
	normalized := provider.Collection{
		ID:          collection.ID,
		Provider:    p.Name(),
		Title:       collection.Title,
		Description: collection.Description,
		License:     collection.License,
		Keywords:    collection.Keywords,
	}

	for _, link := range collection.Links {
		switch link.Rel {
		case "items":
			normalized.ItemURL = link.Href
		case "queryables", "http://www.opengis.net/def/rel/ogc/1.0/queryables":
			normalized.QueryablesURL = link.Href
		}
	}

	if collection.Extent != nil {
		extent := &provider.Extent{}
		if collection.Extent.Spatial != nil {
			extent.Spatial = collection.Extent.Spatial.BBox
		}
		if collection.Extent.Temporal != nil {
			extent.Temporal = collection.Extent.Temporal.Interval
		}
		if len(extent.Spatial) > 0 || len(extent.Temporal) > 0 {
			normalized.Extent = extent
		}
	}

	return normalized
}

// sortedKeys returns map keys in deterministic order.
func sortedKeys[V any](m map[string]V) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
