package copernicus

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/oscarhugopaz/earth-cli/internal/provider"
)

func featureJSON(id string) string {
	return fmt.Sprintf(
		`{"id":%q,"collection":"c","properties":{"datetime":"2026-09-14T00:00:00Z","eo:cloud_cover":1.0},"assets":{},"links":[]}`,
		id,
	)
}

func TestSearchPaginatesUntilLimit(t *testing.T) {
	var requests int
	var secondBody map[string]any
	var p *Provider
	p = newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)

		if body["token"] == nil {
			nextBody, _ := json.Marshal(map[string]any{
				"collections": []string{"c"},
				"limit":       2,
				"token":       "page-2",
			})
			fmt.Fprintf(w,
				`{"features":[%s,%s],"links":[{"rel":"next","method":"POST","href":"%s/search","body":%s}]}`,
				featureJSON("a"), featureJSON("b"), p.BaseURL(), nextBody)
			return
		}

		secondBody = body
		fmt.Fprintf(w, `{"features":[%s,%s],"links":[]}`, featureJSON("c"), featureJSON("d"))
	})

	observations, err := p.Search(context.Background(), provider.SearchRequest{Collection: "c", Limit: 3})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(observations) != 3 {
		t.Fatalf("len(observations) = %d, want 3", len(observations))
	}
	if observations[2].ID != "c" {
		t.Fatalf("third observation = %q", observations[2].ID)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	if secondBody["token"] != "page-2" {
		t.Fatalf("second request body = %v", secondBody)
	}
}

func TestSearchStopsAtLimitWithoutExtraRequest(t *testing.T) {
	var requests int
	var p *Provider
	p = newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		fmt.Fprintf(w,
			`{"features":[%s,%s],"links":[{"rel":"next","method":"POST","href":"%s/search","body":{"token":"x"}}]}`,
			featureJSON("a"), featureJSON("b"), p.BaseURL())
	})

	observations, err := p.Search(context.Background(), provider.SearchRequest{Collection: "c", Limit: 2})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(observations) != 2 {
		t.Fatalf("len = %d, want 2", len(observations))
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}

func TestSearchWithoutLimitDoesNotPaginate(t *testing.T) {
	var requests int
	var p *Provider
	p = newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		fmt.Fprintf(w,
			`{"features":[%s],"links":[{"rel":"next","method":"POST","href":"%s/search","body":{"token":"x"}}]}`,
			featureJSON("a"), p.BaseURL())
	})

	observations, err := p.Search(context.Background(), provider.SearchRequest{Collection: "c"})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(observations) != 1 {
		t.Fatalf("len = %d, want 1", len(observations))
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}

func TestSearchIgnoresOffHostNextLink(t *testing.T) {
	var requests int
	p := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		fmt.Fprintf(w, `{"features":[%s],"links":[{"rel":"next","href":"https://evil.test/search"}]}`, featureJSON("a"))
	})

	observations, err := p.Search(context.Background(), provider.SearchRequest{Collection: "c", Limit: 10})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(observations) != 1 {
		t.Fatalf("len = %d, want 1", len(observations))
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}
