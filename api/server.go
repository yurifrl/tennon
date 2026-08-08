// Package api is Tennon's REST/JSON management surface. The web UI, the CLI, and
// automation are all clients of it. It reads and writes the same store objects
// the controller reconciles, so validation and audit are uniform. It depends
// only on domain and the store port.
package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/yurifrl/tennon/domain"
)

const defaultNamespace = "tennon"

// Server is the HTTP management API.
type Server struct {
	store     domain.Store
	executer  domain.ExecutionCompleter
	log       *slog.Logger
}

// New builds the API server.
func New(store domain.Store, exec domain.ExecutionCompleter, log *slog.Logger) *Server {
	if log == nil {
		log = slog.Default()
	}
	return &Server{store: store, executer: exec, log: log}
}

// Handler returns the API's HTTP handler with all routes mounted.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/v1/connectors", s.listConnectors)
	mux.HandleFunc("POST /api/v1/connectors", s.createConnector)
	mux.HandleFunc("GET /api/v1/connectors/{name}", s.getConnector)
	mux.HandleFunc("DELETE /api/v1/connectors/{name}", s.deleteConnector)
	mux.HandleFunc("GET /api/v1/secrets", s.listSecrets)
	mux.HandleFunc("POST /api/v1/secrets", s.createSecret)
	mux.HandleFunc("GET /api/v1/executions", s.listExecutions)
	mux.HandleFunc("GET /api/v1/executions/{id}", s.getExecution)
	mux.HandleFunc("POST /api/v1/executions/{id}/approve", s.approveExecution)
	mux.HandleFunc("POST /api/v1/executions/{id}/deny", s.denyExecution)
	return mux
}

func (s *Server) listConnectors(w http.ResponseWriter, r *http.Request) {
	resources, err := s.store.List(r.Context(), domain.KindTennonConnector, "")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	out := make([]*domain.Connector, 0, len(resources))
	for _, res := range resources {
		c, err := domain.ConnectorFromResource(res)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		out = append(out, c)
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (s *Server) createConnector(w http.ResponseWriter, r *http.Request) {
	var c domain.Connector
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if c.Meta.Name == "" {
		writeErr(w, http.StatusBadRequest, errors.New("metadata.name is required"))
		return
	}
	if c.Kind == "" {
		c.Kind = domain.KindTennonConnector
	}
	if c.Meta.Namespace == "" {
		c.Meta.Namespace = defaultNamespace
	}
	if c.Spec.RuntimeKind() == "" {
		writeErr(w, http.StatusBadRequest, errors.New("spec.runtime must set exactly one of container/stdio/remote/host"))
		return
	}
	res, err := c.ToResource()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	stored, err := s.store.Apply(r.Context(), res)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	saved, _ := domain.ConnectorFromResource(stored)
	writeJSON(w, http.StatusCreated, saved)
}

func (s *Server) getConnector(w http.ResponseWriter, r *http.Request) {
	res, err := s.store.Get(r.Context(), domain.KindTennonConnector, defaultNamespace, r.PathValue("name"))
	if err != nil {
		s.storeErr(w, err)
		return
	}
	c, _ := domain.ConnectorFromResource(res)
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) deleteConnector(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Delete(r.Context(), domain.KindTennonConnector, defaultNamespace, r.PathValue("name")); err != nil {
		s.storeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// secretRequest is the create-secret payload. Data is write-only.
type secretRequest struct {
	Metadata domain.ObjectMeta `json:"metadata"`
	Data     map[string]string `json:"data"`
}

func (s *Server) createSecret(w http.ResponseWriter, r *http.Request) {
	var req secretRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if req.Metadata.Name == "" {
		writeErr(w, http.StatusBadRequest, errors.New("metadata.name is required"))
		return
	}
	ns := req.Metadata.Namespace
	if ns == "" {
		ns = defaultNamespace
	}
	res, err := domain.NewSecretResource(ns, req.Metadata.Name, req.Data)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if _, err := s.store.Apply(r.Context(), res); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	// Never echo the data back.
	writeJSON(w, http.StatusCreated, map[string]any{
		"metadata": map[string]string{"name": req.Metadata.Name, "namespace": ns},
		"keys":     keysOf(req.Data),
	})
}

func (s *Server) listSecrets(w http.ResponseWriter, r *http.Request) {
	resources, err := s.store.List(r.Context(), domain.KindSecret, "")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	items := make([]map[string]any, 0, len(resources))
	for _, res := range resources {
		data, _ := domain.SecretData(res)
		items = append(items, map[string]any{
			"metadata": map[string]string{"name": res.Meta.Name, "namespace": res.Meta.Namespace},
			"keys":     keysOf(data), // names only, never values
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) storeErr(w http.ResponseWriter, err error) {
	if errors.Is(err, domain.ErrNotFound) {
		writeErr(w, http.StatusNotFound, err)
		return
	}
	writeErr(w, http.StatusInternalServerError, err)
}

func keysOf(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]string{"error": err.Error()})
}

func (s *Server) listExecutions(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	execs, err := s.executer.ListExecutions(r.Context(), state)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": execs})
}

func (s *Server) getExecution(w http.ResponseWriter, r *http.Request) {
	exec, err := s.executer.GetExecution(r.Context(), r.PathValue("id"))
	if err != nil {
		s.storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, exec)
}

func (s *Server) approveExecution(w http.ResponseWriter, r *http.Request) {
	if err := s.executer.ApproveExecution(r.Context(), r.PathValue("id")); err != nil {
		s.storeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) denyExecution(w http.ResponseWriter, r *http.Request) {
	if err := s.executer.DenyExecution(r.Context(), r.PathValue("id")); err != nil {
		s.storeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
