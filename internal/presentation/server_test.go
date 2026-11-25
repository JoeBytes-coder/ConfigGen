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

	"github.com/gin-gonic/gin"

	"configgen/internal/domain"
)

type mockUseCase struct {
	planFn   func(domain.Selection) (domain.Plan, error)
	renderFn func(domain.Selection) (domain.Record, error)
	getFn    func(int64) (domain.Record, error)
	listFn   func(int, int) ([]domain.RecordSummary, int, error)
	healthFn func() error
}

func (m *mockUseCase) Plan(sel domain.Selection) (domain.Plan, error) { return m.planFn(sel) }
func (m *mockUseCase) Render(sel domain.Selection) (domain.Record, error) {
	return m.renderFn(sel)
}
func (m *mockUseCase) GetRecord(id int64) (domain.Record, error) { return m.getFn(id) }
func (m *mockUseCase) ListRecords(offset, limit int) ([]domain.RecordSummary, int, error) {
	return m.listFn(offset, limit)
}
func (m *mockUseCase) Health() error { return m.healthFn() }

func init() { gin.SetMode(gin.TestMode) }

func newTestRouter(m *mockUseCase) *gin.Engine { return NewServer(m).router }

func do(router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	return w
}

func TestCatalogAndHealth(t *testing.T) {
	router := newTestRouter(&mockUseCase{healthFn: func() error { return nil }})

	w := do(router, "GET", "/api/v1/catalog", "")
	if w.Code != http.StatusOK {
		t.Fatalf("catalog status %d", w.Code)
	}
	var catalog domain.CatalogPayload
	if err := json.Unmarshal(w.Body.Bytes(), &catalog); err != nil {
		t.Fatal(err)
	}
	if len(catalog.Kinds) != len(domain.Catalog) {
		t.Errorf("expected %d kinds, got %d", len(domain.Catalog), len(catalog.Kinds))
	}
	if catalog.Version == "" {
		t.Error("catalog version missing")
	}

	w = do(router, "GET", "/health", "")
	if w.Code != http.StatusOK {
		t.Errorf("health status %d", w.Code)
	}
}

func TestHealthDegraded(t *testing.T) {
	router := newTestRouter(&mockUseCase{healthFn: func() error { return errors.New("db down") }})
	if w := do(router, "GET", "/health", ""); w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestPlanEndpoint(t *testing.T) {
	plan := domain.Plan{
		Type:        domain.TypeK8s,
		App:         domain.AppContext{Name: "demo"},
		Nodes:       []domain.Node{{Kind: "Deployment", Name: "demo-deployment", Level: 2}},
		Diagnostics: []domain.Diagnostic{},
		Valid:       true, CatalogVersion: domain.CatalogVersion,
	}
	router := newTestRouter(&mockUseCase{
		planFn: func(sel domain.Selection) (domain.Plan, error) {
			if sel.Type != domain.TypeK8s || sel.App.Name != "demo" {
				t.Errorf("unexpected selection: %+v", sel)
			}
			return plan, nil
		},
	})
	w := do(router, "POST", "/api/v1/plan", `{"type":"k8s","app":{"name":"demo"},"resources":["Deployment"]}`)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "demo-deployment") {
		t.Error("plan response missing node")
	}
}

func TestPlanValidationError(t *testing.T) {
	router := newTestRouter(&mockUseCase{
		planFn: func(domain.Selection) (domain.Plan, error) {
			return domain.Plan{}, domain.NewBadRequest("配置校验失败",
				domain.Diagnostic{Severity: domain.SeverityError, Code: "invalid_app_name", Message: "bad"})
		},
	})
	w := do(router, "POST", "/api/v1/plan", `{"type":"k8s","app":{"name":"X"},"resources":["Deployment"]}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "invalid_app_name") {
		t.Error("expected diagnostics in response")
	}
}

func TestPlanConflictReturns422(t *testing.T) {
	router := newTestRouter(&mockUseCase{
		planFn: func(domain.Selection) (domain.Plan, error) {
			return domain.Plan{}, domain.NewUnprocessable("资源依赖无法满足")
		},
	})
	w := do(router, "POST", "/api/v1/plan", `{"type":"k8s","app":{"name":"demo"},"resources":["Deployment"]}`)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", w.Code)
	}
}

func TestRenderEndpoint(t *testing.T) {
	rec := domain.Record{
		ID: 7, Type: domain.TypeK8s,
		Files:     []domain.Artifact{{Path: "deployment.yaml", Kind: "Deployment", Content: "kind: Deployment"}},
		Bundle:    "---\nkind: Deployment\n",
		CreatedAt: time.Now(),
	}
	router := newTestRouter(&mockUseCase{
		renderFn: func(domain.Selection) (domain.Record, error) { return rec, nil },
	})
	w := do(router, "POST", "/api/v1/render", `{"type":"k8s","app":{"name":"demo"},"resources":["Deployment"]}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "deployment.yaml") {
		t.Error("response missing files")
	}
}

func TestUnknownFieldRejected(t *testing.T) {
	router := newTestRouter(&mockUseCase{})
	w := do(router, "POST", "/api/v1/plan", `{"type":"k8s","app":{"name":"demo"},"unexpected":true}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for unknown field, got %d", w.Code)
	}
}

func TestListAndGetRecord(t *testing.T) {
	router := newTestRouter(&mockUseCase{
		listFn: func(offset, limit int) ([]domain.RecordSummary, int, error) {
			return []domain.RecordSummary{{ID: 1, Type: domain.TypeK8s, AppName: "demo"}}, 1, nil
		},
		getFn: func(id int64) (domain.Record, error) {
			if id == 404 {
				return domain.Record{}, sql.ErrNoRows
			}
			return domain.Record{ID: id, Type: domain.TypeK8s}, nil
		},
	})
	w := do(router, "GET", "/api/v1/records?offset=0&limit=10", "")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"total":1`) {
		t.Errorf("list response unexpected: %d %s", w.Code, w.Body.String())
	}

	if w := do(router, "GET", "/api/v1/records/3", ""); w.Code != http.StatusOK {
		t.Errorf("get expected 200, got %d", w.Code)
	}
	if w := do(router, "GET", "/api/v1/records/404", ""); w.Code != http.StatusNotFound {
		t.Errorf("get expected 404, got %d", w.Code)
	}
	if w := do(router, "GET", "/api/v1/records/abc", ""); w.Code != http.StatusBadRequest {
		t.Errorf("get expected 400, got %d", w.Code)
	}
}
