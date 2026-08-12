package domain

import (
	"testing"
)

type mockToolProvider struct {
	tools []Tool
}

func (m *mockToolProvider) Tools() []Tool {
	return m.tools
}

func TestToolCatalog_Search(t *testing.T) {
	provider := &mockToolProvider{
		tools: []Tool{
			{
				Name:        "grafana.query",
				Description: "Query Grafana metrics and dashboards",
				Connector:   "grafana",
				Tags:        []string{"observability", "metrics"},
			},
			{
				Name:        "prometheus.query",
				Description: "Execute PromQL queries against Prometheus",
				Connector:   "prometheus",
				Tags:        []string{"observability", "metrics"},
			},
			{
				Name:        "slack.send_message",
				Description: "Send a message to a Slack channel",
				Connector:   "slack",
				Tags:        []string{"communication"},
			},
		},
	}

	catalog := NewToolCatalog(provider)

	tests := []struct {
		name          string
		query         string
		expectedCount int
		expectFirst   string // expected first result name
	}{
		{
			name:          "exact name match",
			query:         "grafana.query",
			expectedCount: 1,
			expectFirst:   "grafana.query",
		},
		{
			name:          "connector match",
			query:         "grafana",
			expectedCount: 1,
			expectFirst:   "grafana.query",
		},
		{
			name:          "tag match",
			query:         "observability",
			expectedCount: 2,
		},
		{
			name:          "description keyword",
			query:         "query",
			expectedCount: 2,
		},
		{
			name:          "no match",
			query:         "nonexistent",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := catalog.Search(SearchRequest{
				Query: tt.query,
				Limit: 20,
			})

			if len(results) != tt.expectedCount {
				t.Errorf("expected %d results, got %d", tt.expectedCount, len(results))
			}

			if tt.expectFirst != "" && len(results) > 0 {
				if results[0].Tool.Name != tt.expectFirst {
					t.Errorf("expected first result %s, got %s", tt.expectFirst, results[0].Tool.Name)
				}
			}

			// Verify results are sorted by score descending
			for i := 1; i < len(results); i++ {
				if results[i].Score > results[i-1].Score {
					t.Errorf("results not sorted by score: result %d score %f > result %d score %f",
						i, results[i].Score, i-1, results[i-1].Score)
				}
			}
		})
	}
}

func TestToolCatalog_SearchLimit(t *testing.T) {
	// Create many tools
	var tools []Tool
	for i := 0; i < 50; i++ {
		tools = append(tools, Tool{
			Name:        "tool" + string(rune(i)),
			Description: "test tool for search",
			Connector:   "test",
		})
	}

	provider := &mockToolProvider{tools: tools}
	catalog := NewToolCatalog(provider)

	results := catalog.Search(SearchRequest{
		Query: "test",
		Limit: 10,
	})

	if len(results) != 10 {
		t.Errorf("expected limit of 10 results, got %d", len(results))
	}
}

func TestToolCatalog_ListAll(t *testing.T) {
	provider := &mockToolProvider{
		tools: []Tool{
			{Name: "tool1", Connector: "conn1"},
			{Name: "tool2", Connector: "conn2"},
		},
	}

	catalog := NewToolCatalog(provider)
	all := catalog.ListAll()

	if len(all) != 2 {
		t.Errorf("expected 2 tools, got %d", len(all))
	}
}
