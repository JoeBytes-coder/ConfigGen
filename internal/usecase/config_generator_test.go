package usecase

import (
	"errors"
	"testing"
	"time"

	"configgen/internal/domain"
)

type mockGenerator struct {
	genFunc func(domain.ConfigRequest) (domain.ConfigResult, error)
}

func (m *mockGenerator) Generate(req domain.ConfigRequest) (domain.ConfigResult, error) {
	return m.genFunc(req)
}

type mockStore struct {
	saveFunc func(domain.ConfigRecord) (int64, error)
	findFunc func(int64) (domain.ConfigRecord, error)
	listFunc func(int, int) ([]domain.ConfigRecord, error)
}

func (m *mockStore) Save(rec domain.ConfigRecord) (int64, error) {
	return m.saveFunc(rec)
}

func (m *mockStore) Find(id int64) (domain.ConfigRecord, error) {
	return m.findFunc(id)
}

func (m *mockStore) List(offset, limit int) ([]domain.ConfigRecord, error) {
	return m.listFunc(offset, limit)
}

func TestPrepare(t *testing.T) {
	t.Run("non-k8s type does nothing", func(t *testing.T) {
		cg := NewConfigGenerator(nil, nil)
		req := &domain.ConfigRequest{Type: "compose", Image: "", Tag: ""}
		cg.Prepare(req)
		if req.Image != "" || req.Tag != "" {
			t.Error("expected no defaults applied for non-k8s type")
		}
	})

	t.Run("k8s with no resources does nothing", func(t *testing.T) {
		cg := NewConfigGenerator(nil, nil)
		req := &domain.ConfigRequest{Type: "k8s", K8sResource: nil, Image: "", Tag: ""}
		cg.Prepare(req)
		if req.Image != "" || req.Tag != "" {
			t.Error("expected no defaults when K8sResource is nil")
		}
	})

	t.Run("k8s with workload sets image defaults", func(t *testing.T) {
		cg := NewConfigGenerator(nil, nil)
		req := &domain.ConfigRequest{Type: "k8s", K8sResource: []string{"Deployment"}, Image: "", Tag: "", Port: 0}
		cg.Prepare(req)
		if req.Image != "nginx" {
			t.Errorf("expected image nginx, got %s", req.Image)
		}
		if req.Tag != "latest" {
			t.Errorf("expected tag latest, got %s", req.Tag)
		}
		if req.Port != 0 {
			t.Errorf("expected port 0 for workload only, got %d", req.Port)
		}
	})

	t.Run("k8s with service sets port", func(t *testing.T) {
		cg := NewConfigGenerator(nil, nil)
		req := &domain.ConfigRequest{Type: "k8s", K8sResource: []string{"Service"}, Image: "", Tag: "", Port: 0}
		cg.Prepare(req)
		if req.Image != "nginx" {
			t.Errorf("expected image nginx, got %s", req.Image)
		}
		if req.Port != 80 {
			t.Errorf("expected port 80, got %d", req.Port)
		}
	})

	t.Run("existing values not overwritten", func(t *testing.T) {
		cg := NewConfigGenerator(nil, nil)
		req := &domain.ConfigRequest{Type: "k8s", K8sResource: []string{"Deployment"}, Image: "myimage", Tag: "v1", Port: 8080}
		cg.Prepare(req)
		if req.Image != "myimage" {
			t.Errorf("expected image myimage, got %s", req.Image)
		}
		if req.Port != 8080 {
			t.Errorf("expected port 8080, got %d", req.Port)
		}
	})

	t.Run("only ConfigMap and Secret does NOT set port", func(t *testing.T) {
		cg := NewConfigGenerator(nil, nil)
		req := &domain.ConfigRequest{Type: "k8s", K8sResource: []string{"ConfigMap", "Secret"}, Image: "", Tag: "", Port: 0}
		cg.Prepare(req)
		if req.Image != "" || req.Tag != "" {
			t.Error("expected no image defaults for config-only resources")
		}
		if req.Port != 0 {
			t.Errorf("expected port 0 for config-only resources, got %d", req.Port)
		}
	})
}

func TestGenerateAndSave(t *testing.T) {
	t.Run("successful generation and save", func(t *testing.T) {
		gen := &mockGenerator{
			genFunc: func(req domain.ConfigRequest) (domain.ConfigResult, error) {
				return domain.ConfigResult{Type: "compose", Config: "version: '3'"}, nil
			},
		}
		store := &mockStore{
			saveFunc: func(rec domain.ConfigRecord) (int64, error) {
				if rec.Result.Config != "version: '3'" {
					t.Error("unexpected config saved")
				}
				return 42, nil
			},
		}
		cg := NewConfigGenerator(gen, store)
		rec, err := cg.GenerateAndSave(domain.ConfigRequest{Type: "compose", AppName: "test", Image: "n", Tag: "l", Port: 80})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rec.ID != 42 {
			t.Errorf("expected id 42, got %d", rec.ID)
		}
		if rec.Result.Config != "version: '3'" {
			t.Errorf("unexpected config: %s", rec.Result.Config)
		}
	})

	t.Run("generator error propagates", func(t *testing.T) {
		gen := &mockGenerator{
			genFunc: func(req domain.ConfigRequest) (domain.ConfigResult, error) {
				return domain.ConfigResult{}, errors.New("gen failed")
			},
		}
		store := &mockStore{
			saveFunc: func(rec domain.ConfigRecord) (int64, error) {
				t.Error("save should not be called on generator error")
				return 0, nil
			},
		}
		cg := NewConfigGenerator(gen, store)
		_, err := cg.GenerateAndSave(domain.ConfigRequest{Type: "compose", AppName: "test", Image: "n", Tag: "l", Port: 80})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("store error propagates", func(t *testing.T) {
		gen := &mockGenerator{
			genFunc: func(req domain.ConfigRequest) (domain.ConfigResult, error) {
				return domain.ConfigResult{Type: "compose", Config: "data"}, nil
			},
		}
		store := &mockStore{
			saveFunc: func(rec domain.ConfigRecord) (int64, error) {
				return 0, errors.New("db error")
			},
		}
		cg := NewConfigGenerator(gen, store)
		_, err := cg.GenerateAndSave(domain.ConfigRequest{Type: "compose", AppName: "test", Image: "n", Tag: "l", Port: 80})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("created at is set", func(t *testing.T) {
		gen := &mockGenerator{
			genFunc: func(req domain.ConfigRequest) (domain.ConfigResult, error) {
				return domain.ConfigResult{Type: "compose", Config: "data"}, nil
			},
		}
		before := time.Now().UTC()
		store := &mockStore{
			saveFunc: func(rec domain.ConfigRecord) (int64, error) {
				if rec.CreatedAt.Before(before) {
					t.Error("created_at should not be before test start")
				}
				return 1, nil
			},
		}
		cg := NewConfigGenerator(gen, store)
		_, err := cg.GenerateAndSave(domain.ConfigRequest{Type: "compose", AppName: "test", Image: "n", Tag: "l", Port: 80})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestGetRecord(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		expected := domain.ConfigRecord{ID: 1, CreatedAt: time.Now()}
		store := &mockStore{
			findFunc: func(id int64) (domain.ConfigRecord, error) {
				return expected, nil
			},
		}
		cg := NewConfigGenerator(nil, store)
		rec, err := cg.GetRecord(1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rec.ID != 1 {
			t.Errorf("expected id 1, got %d", rec.ID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		store := &mockStore{
			findFunc: func(id int64) (domain.ConfigRecord, error) {
				return domain.ConfigRecord{}, errors.New("not found")
			},
		}
		cg := NewConfigGenerator(nil, store)
		_, err := cg.GetRecord(99)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestListRecords(t *testing.T) {
	t.Run("empty list", func(t *testing.T) {
		store := &mockStore{
			listFunc: func(offset, limit int) ([]domain.ConfigRecord, error) {
				return []domain.ConfigRecord{}, nil
			},
		}
		cg := NewConfigGenerator(nil, store)
		recs, err := cg.ListRecords(0, 20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(recs) != 0 {
			t.Errorf("expected 0 records, got %d", len(recs))
		}
	})

	t.Run("paginated results", func(t *testing.T) {
		store := &mockStore{
			listFunc: func(offset, limit int) ([]domain.ConfigRecord, error) {
				if offset != 0 || limit != 10 {
					t.Errorf("unexpected pagination: offset=%d limit=%d", offset, limit)
				}
				return []domain.ConfigRecord{
					{ID: 1}, {ID: 2}, {ID: 3},
				}, nil
			},
		}
		cg := NewConfigGenerator(nil, store)
		recs, err := cg.ListRecords(0, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(recs) != 3 {
			t.Errorf("expected 3 records, got %d", len(recs))
		}
	})
}
