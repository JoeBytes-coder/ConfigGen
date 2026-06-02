package presentation

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"configgen/internal/domain"

	"github.com/gin-gonic/gin"
)

type mockConfigGenerator struct {
	prepareFunc       func(*domain.ConfigRequest)
	generateAndSaveFn func(domain.ConfigRequest) (domain.ConfigRecord, error)
	getRecordFn       func(int64) (domain.ConfigRecord, error)
	listRecordsFn     func(int, int) ([]domain.ConfigRecord, error)
}

func (m *mockConfigGenerator) Prepare(req *domain.ConfigRequest) {
	if m.prepareFunc != nil {
		m.prepareFunc(req)
	}
}

func (m *mockConfigGenerator) GenerateAndSave(req domain.ConfigRequest) (domain.ConfigRecord, error) {
	return m.generateAndSaveFn(req)
}

func (m *mockConfigGenerator) GetRecord(id int64) (domain.ConfigRecord, error) {
	return m.getRecordFn(id)
}

func (m *mockConfigGenerator) ListRecords(offset, limit int) ([]domain.ConfigRecord, error) {
	return m.listRecordsFn(offset, limit)
}

// matchUsecaseInterface ensures mockConfigGenerator satisfies the interface.
var _ = &mockConfigGenerator{}

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestServer(mock *mockConfigGenerator) *Server {
	g := gin.New()
	s := &Server{router: g, configGen: mock}

	g.GET("/health", s.health)
	g.POST("/api/v1/generate", s.generate)
	g.POST("/api/v1/generate/:type", s.generateWithType)
	g.POST("/api/v1/generate/compose", s.generateCompose)
	g.POST("/api/v1/generate/k8s", s.generateK8s)
	g.GET("/api/v1/configs", s.listConfigs)
	g.GET("/api/v1/configs/:id", s.getConfig)
	g.GET("/api/v1/configs/:id/resources", s.getConfigResources)
	g.GET("/api/v1/configs/:id/edges", s.getConfigEdges)
	g.GET("/api/v1/configs/:id/graph", s.getConfigGraph)

	return s
}

func TestHealth(t *testing.T) {
	s := newTestServer(&mockConfigGenerator{})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	s.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var body map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["status"] != "ok" {
		t.Errorf("expected status ok, got %s", body["status"])
	}
}

func TestGenerate(t *testing.T) {
	t.Run("valid compose request returns 201", func(t *testing.T) {
		mock := &mockConfigGenerator{
			prepareFunc: func(req *domain.ConfigRequest) {},
			generateAndSaveFn: func(req domain.ConfigRequest) (domain.ConfigRecord, error) {
				return domain.ConfigRecord{
					ID: 1,
					Request: domain.ConfigRequest{
						Type: "compose", AppName: "test", Image: "n", Tag: "l", Port: 80,
					},
					Result:    domain.ConfigResult{Type: "compose", Config: "version: '3'"},
					CreatedAt: time.Now(),
				}, nil
			},
		}
		s := newTestServer(mock)
		w := httptest.NewRecorder()
		body := `{"type":"compose","app_name":"test","image":"n","tag":"l","port":80}`
		req, _ := http.NewRequest("POST", "/api/v1/generate", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		s.router.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
		var rec domain.ConfigRecord
		if err := json.Unmarshal(w.Body.Bytes(), &rec); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		if rec.ID != 1 {
			t.Errorf("expected id 1, got %d", rec.ID)
		}
	})

	t.Run("missing type defaults to k8s", func(t *testing.T) {
		mock := &mockConfigGenerator{
			prepareFunc: func(req *domain.ConfigRequest) {},
			generateAndSaveFn: func(req domain.ConfigRequest) (domain.ConfigRecord, error) {
				if req.Type != "k8s" {
					t.Errorf("expected type k8s, got %s", req.Type)
				}
				return domain.ConfigRecord{ID: 2}, nil
			},
		}
		s := newTestServer(mock)
		w := httptest.NewRecorder()
		body := `{"app_name":"test","image":"n","tag":"l","port":80}`
		req, _ := http.NewRequest("POST", "/api/v1/generate", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		s.router.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		mock := &mockConfigGenerator{
			prepareFunc: func(req *domain.ConfigRequest) {},
		}
		s := newTestServer(mock)
		w := httptest.NewRecorder()
		body := `{"type":"compose","app_name":"","image":"","tag":"","port":0}`
		req, _ := http.NewRequest("POST", "/api/v1/generate", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		s.router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("generator error returns 500", func(t *testing.T) {
		mock := &mockConfigGenerator{
			prepareFunc: func(req *domain.ConfigRequest) {},
			generateAndSaveFn: func(req domain.ConfigRequest) (domain.ConfigRecord, error) {
				return domain.ConfigRecord{}, errors.New("gen failed")
			},
		}
		s := newTestServer(mock)
		w := httptest.NewRecorder()
		body := `{"type":"compose","app_name":"test","image":"n","tag":"l","port":80}`
		req, _ := http.NewRequest("POST", "/api/v1/generate", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		s.router.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid JSON body returns 400", func(t *testing.T) {
		mock := &mockConfigGenerator{}
		s := newTestServer(mock)
		w := httptest.NewRecorder()
		body := `not json`
		req, _ := http.NewRequest("POST", "/api/v1/generate", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		s.router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestGenerateWithType(t *testing.T) {
	t.Run("compose endpoint forces type", func(t *testing.T) {
		mock := &mockConfigGenerator{
			prepareFunc: func(req *domain.ConfigRequest) {},
			generateAndSaveFn: func(req domain.ConfigRequest) (domain.ConfigRecord, error) {
				if req.Type != "compose" {
					t.Errorf("expected type compose, got %s", req.Type)
				}
				return domain.ConfigRecord{ID: 3}, nil
			},
		}
		s := newTestServer(mock)
		w := httptest.NewRecorder()
		body := `{"app_name":"test","image":"n","tag":"l","port":80}`
		req, _ := http.NewRequest("POST", "/api/v1/generate/compose", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		s.router.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("url param type is used", func(t *testing.T) {
		mock := &mockConfigGenerator{
			prepareFunc: func(req *domain.ConfigRequest) {},
			generateAndSaveFn: func(req domain.ConfigRequest) (domain.ConfigRecord, error) {
				if req.Type != "dockerfile" {
					t.Errorf("expected type dockerfile, got %s", req.Type)
				}
				return domain.ConfigRecord{ID: 4}, nil
			},
		}
		s := newTestServer(mock)
		w := httptest.NewRecorder()
		body := `{"app_name":"test","image":"n","tag":"l","port":80}`
		req, _ := http.NewRequest("POST", "/api/v1/generate/dockerfile", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		s.router.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestGetConfig(t *testing.T) {
	t.Run("found returns record", func(t *testing.T) {
		mock := &mockConfigGenerator{
			getRecordFn: func(id int64) (domain.ConfigRecord, error) {
				return domain.ConfigRecord{ID: id, CreatedAt: time.Now()}, nil
			},
		}
		s := newTestServer(mock)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/configs/1", nil)
		s.router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("not found returns 404", func(t *testing.T) {
		mock := &mockConfigGenerator{
			getRecordFn: func(id int64) (domain.ConfigRecord, error) {
				return domain.ConfigRecord{}, sql.ErrNoRows
			},
		}
		s := newTestServer(mock)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/configs/99", nil)
		s.router.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		mock := &mockConfigGenerator{}
		s := newTestServer(mock)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/configs/abc", nil)
		s.router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestListConfigs(t *testing.T) {
	mock := &mockConfigGenerator{
		listRecordsFn: func(offset, limit int) ([]domain.ConfigRecord, error) {
			return []domain.ConfigRecord{}, nil
		},
	}
	s := newTestServer(mock)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/configs", nil)
	s.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGraphEndpoints(t *testing.T) {
	rec := domain.ConfigRecord{
		ID: 1,
		Result: domain.ConfigResult{
			Type: "k8s",
			Resources: []domain.ResourceNode{
				{Kind: "Deployment", Name: "app-deployment"},
				{Kind: "Service", Name: "app-service"},
			},
			Edges: []domain.ResourceEdge{
				{SourceKind: "Service", SourceName: "app-service", TargetKind: "Deployment", TargetName: "app-deployment", Relation: "selects"},
			},
		},
		CreatedAt: time.Now(),
	}

	t.Run("graph returns nodes and edges", func(t *testing.T) {
		mock := &mockConfigGenerator{
			getRecordFn: func(id int64) (domain.ConfigRecord, error) {
				return rec, nil
			},
		}
		s := newTestServer(mock)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/configs/1/graph", nil)
		s.router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var body map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &body)
		resources, ok := body["resources"].([]interface{})
		if !ok || len(resources) != 2 {
			t.Errorf("expected 2 resources, got %v", body["resources"])
		}
		edges, ok := body["edges"].([]interface{})
		if !ok || len(edges) != 1 {
			t.Errorf("expected 1 edge, got %v", body["edges"])
		}
	})

	t.Run("resources endpoint", func(t *testing.T) {
		mock := &mockConfigGenerator{
			getRecordFn: func(id int64) (domain.ConfigRecord, error) {
				return rec, nil
			},
		}
		s := newTestServer(mock)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/configs/1/resources", nil)
		s.router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("edges endpoint", func(t *testing.T) {
		mock := &mockConfigGenerator{
			getRecordFn: func(id int64) (domain.ConfigRecord, error) {
				return rec, nil
			},
		}
		s := newTestServer(mock)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/configs/1/edges", nil)
		s.router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})
}
