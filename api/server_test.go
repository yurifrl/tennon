package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yurifrl/tennon/domain"
	"github.com/yurifrl/tennon/store"
)

// mockToolProvider implements domain.ToolProvider for testing.
type mockToolProvider struct{}

func (m *mockToolProvider) Tools() []domain.Tool { return nil }

// mockExecuter implements domain.ExecutionCompleter for testing.
type mockExecuter struct{}

func (m *mockExecuter) ApproveExecution(ctx context.Context, execID string) error { return nil }
func (m *mockExecuter) DenyExecution(ctx context.Context, execID string) error    { return nil }
func (m *mockExecuter) ListExecutions(ctx context.Context, state string) ([]*domain.Execution, error) {
	return nil, nil
}
func (m *mockExecuter) GetExecution(ctx context.Context, execID string) (*domain.Execution, error) {
	return nil, nil
}

func TestCreateConnector_KindNormalization(t *testing.T) {
	st := store.NewMemory()
	tokenSvc := domain.NewTokenService(st)
	catalog := domain.NewToolCatalog(&mockToolProvider{})
	srv := New(st, &mockExecuter{}, tokenSvc, catalog, nil)
	handler := srv.Handler()

	tests := []struct {
		name       string
		kind       string
		wantStatus int
		wantInList bool
	}{
		{
			name:       "legacy alias Connector",
			kind:       "Connector",
			wantStatus: http.StatusCreated,
			wantInList: true,
		},
		{
			name:       "canonical TennonConnector",
			kind:       "TennonConnector",
			wantStatus: http.StatusCreated,
			wantInList: true,
		},
		{
			name:       "empty kind",
			kind:       "",
			wantStatus: http.StatusCreated,
			wantInList: true,
		},
		{
			name:       "invalid kind",
			kind:       "BogusKind",
			wantStatus: http.StatusBadRequest,
			wantInList: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create connector with specified kind
			payload := map[string]any{
				"kind": tt.kind,
				"metadata": map[string]string{
					"name": "test-" + tt.name,
				},
				"spec": map[string]any{
					"runtime": map[string]any{
						"remote": map[string]string{"url": "https://example.com/mcp"},
					},
					"transport": "stdio",
				},
			}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequest("POST", "/api/v1/connectors", bytes.NewReader(body))
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("POST status = %d, want %d; body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}

			if !tt.wantInList {
				// For invalid kinds, we expect a 400 and no list entry
				return
			}

			// Verify the connector appears in the list
			listReq := httptest.NewRequest("GET", "/api/v1/connectors", nil)
			listRec := httptest.NewRecorder()
			handler.ServeHTTP(listRec, listReq)

			if listRec.Code != http.StatusOK {
				t.Fatalf("GET list status = %d, want %d", listRec.Code, http.StatusOK)
			}

			var listResp struct {
				Items []domain.Connector `json:"items"`
			}
			if err := json.NewDecoder(listRec.Body).Decode(&listResp); err != nil {
				t.Fatalf("decode list: %v", err)
			}

			// Find our connector in the list
			found := false
			for _, c := range listResp.Items {
				if c.Meta.Name == "test-"+tt.name {
					found = true
					if c.Kind != domain.KindTennonConnector {
						t.Errorf("stored kind = %q, want %q", c.Kind, domain.KindTennonConnector)
					}
					break
				}
			}
			if !found {
				t.Errorf("connector %q not found in list", "test-"+tt.name)
			}
		})
	}
}
