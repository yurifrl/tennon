package domain

import (
	"sort"
	"strings"
)

// ToolCatalog provides search and discovery over registered tools.
type ToolCatalog struct {
	toolProvider ToolProvider
}

// ToolProvider is a port for getting the current set of exposed tools.
type ToolProvider interface {
	Tools() []Tool
}

// NewToolCatalog creates a catalog that indexes tools from a provider.
func NewToolCatalog(provider ToolProvider) *ToolCatalog {
	return &ToolCatalog{toolProvider: provider}
}

// SearchRequest specifies search parameters.
type SearchRequest struct {
	Query string
	Limit int
}

// SearchResult represents a matched tool with its relevance score.
type SearchResult struct {
	Tool  Tool
	Score float64
}

// Search finds tools by intent using keyword/substring matching over name, description, and tags.
func (c *ToolCatalog) Search(req SearchRequest) []SearchResult {
	if req.Query == "" {
		return nil
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}

	query := strings.ToLower(req.Query)
	tools := c.toolProvider.Tools()
	
	var results []SearchResult
	for _, tool := range tools {
		score := c.scoreMatch(query, tool)
		if score > 0 {
			results = append(results, SearchResult{
				Tool:  tool,
				Score: score,
			})
		}
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Limit results
	if len(results) > req.Limit {
		results = results[:req.Limit]
	}

	return results
}

// scoreMatch computes a relevance score for a tool against a query.
// Returns 0 if no match, higher scores for better matches.
func (c *ToolCatalog) scoreMatch(query string, tool Tool) float64 {
	var score float64

	name := strings.ToLower(tool.Name)
	description := strings.ToLower(tool.Description)
	connector := strings.ToLower(tool.Connector)

	// Exact name match: highest score
	if name == query {
		score += 100.0
	} else if strings.Contains(name, query) {
		// Name substring: high score
		score += 50.0
	}

	// Connector match
	if connector == query || strings.Contains(connector, query) {
		score += 30.0
	}

	// Description match
	if strings.Contains(description, query) {
		// Boost if query appears near the beginning
		idx := strings.Index(description, query)
		if idx >= 0 {
			positionBoost := 20.0 * (1.0 - float64(idx)/float64(len(description)))
			score += positionBoost
		}
	}

	// Tag matches
	for _, tag := range tool.Tags {
		tagLower := strings.ToLower(tag)
		if tagLower == query {
			score += 40.0
		} else if strings.Contains(tagLower, query) {
			score += 20.0
		}
	}

	return score
}

// ListAll returns all registered tools (no search filter).
func (c *ToolCatalog) ListAll() []Tool {
	return c.toolProvider.Tools()
}
