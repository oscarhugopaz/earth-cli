package cli

import (
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/oscarhugopaz/earth-cli/internal/provider"
)

func newCollectionsCommand(env Environment) *cobra.Command {
	var (
		search string
		limit  int
	)

	cmd := &cobra.Command{
		Use:   "collections",
		Short: "List Earth observation collections",
		Long: `List the collections exposed by the selected provider's STAC catalog.

Filtering happens client-side, so both collection ids and titles are searched:

  earth collections --search sentinel`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			session, err := newSession(cmd, env)
			if err != nil {
				return err
			}
			p, err := session.selectedProvider()
			if err != nil {
				return err
			}

			// A text filter must see the whole catalog, so fetch everything
			// and apply --limit to the filtered result instead.
			fetchLimit := limit
			if strings.TrimSpace(search) != "" {
				fetchLimit = 0
			}

			collections, err := p.Collections(cmd.Context(), fetchLimit)
			if err != nil {
				return err
			}
			if strings.TrimSpace(search) != "" {
				collections = filterCollections(collections, search)
				if limit > 0 && len(collections) > limit {
					collections = collections[:limit]
				}
			}

			if session.json {
				return session.printer.JSONValue(collections)
			}
			if len(collections) == 0 {
				session.printer.Line("No collections found.")
				return nil
			}

			rows := make([][]string, 0, len(collections))
			for _, collection := range collections {
				rows = append(rows, []string{collection.ID, collection.Title})
			}
			session.printer.Table([]string{"ID", "TITLE"}, rows)
			return nil
		},
	}

	cmd.Flags().StringVar(&search, "search", "", "filter collections client-side by text")
	cmd.Flags().IntVar(&limit, "limit", 0, "maximum number of collections to return (0 = all)")
	return cmd
}

// filterCollections matches every query term and ranks results by relevance so
// an id or title match outranks a keyword match. This keeps
// "earth collections --search sentinel" focused on sentinel-* products.
func filterCollections(collections []provider.Collection, query string) []provider.Collection {
	terms := strings.Fields(strings.ToLower(query))
	type scoredCollection struct {
		collection provider.Collection
		score      int
	}

	matches := make([]scoredCollection, 0, len(collections))
	for _, collection := range collections {
		id := strings.ToLower(collection.ID)
		title := strings.ToLower(collection.Title)
		keywords := strings.ToLower(strings.Join(collection.Keywords, " "))

		score := 0
		matched := true
		for _, term := range terms {
			termScore := 0
			switch {
			case strings.HasPrefix(id, term):
				termScore = 100
			case strings.Contains(id, term):
				termScore = 60
			case strings.Contains(title, term):
				termScore = 40
			case strings.Contains(keywords, term):
				termScore = 20
			default:
				matched = false
			}
			if !matched {
				break
			}
			score += termScore
		}
		if matched {
			matches = append(matches, scoredCollection{collection: collection, score: score})
		}
	}

	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		return matches[i].collection.ID < matches[j].collection.ID
	})

	filtered := make([]provider.Collection, 0, len(matches))
	for _, match := range matches {
		filtered = append(filtered, match.collection)
	}
	return filtered
}
