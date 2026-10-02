package copernicus

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/oscarhugopaz/earth-cli/internal/provider"
)

// Item fetches a single item and returns its normalized form including asset
// details.
func (p *Provider) Item(ctx context.Context, collection, id string) (provider.Observation, error) {
	collection = strings.TrimSpace(collection)
	id = strings.TrimSpace(id)
	if collection == "" || id == "" {
		return provider.Observation{}, fmt.Errorf("collection and item id are required")
	}

	endpoint := p.baseURL + "/collections/" + encodePathSegment(collection) + "/items/" + encodePathSegment(id)
	var feature stacFeature
	if err := p.do(ctx, http.MethodGet, endpoint, nil, &feature); err != nil {
		var responseErr *provider.ResponseError
		if errors.As(err, &responseErr) && responseErr.Code == http.StatusNotFound {
			return provider.Observation{}, fmt.Errorf("item %q was not found in collection %q", id, collection)
		}
		return provider.Observation{}, err
	}

	return normalizeObservation(feature, collection, p, true), nil
}
