package storage

import (
	"path/filepath"
	"testing"
	"time"

	"configgen/internal/domain"
)

func newRecord(name string) domain.Record {
	return domain.Record{
		Type: domain.TypeK8s,
		App:  domain.AppContext{Name: name, Namespace: "prod"},
		Selection: domain.Selection{
			Type: domain.TypeK8s, App: domain.AppContext{Name: name}, Resources: []string{"Deployment"},
		},
		Plan: domain.Plan{
			Type: domain.TypeK8s, App: domain.AppContext{Name: name},
			Nodes: []domain.Node{{Kind: "Deployment", Name: name + "-deployment", Level: 2}},
		},
		Files:          []domain.Artifact{{Path: "deployment.yaml", Kind: "Deployment", Content: "kind: Deployment\n"}},
		Bundle:         "---\nkind: Deployment\n",
		CatalogVersion: domain.CatalogVersion,
		CreatedAt:      time.Now().UTC().Truncate(time.Second),
	}
}

func TestSQLiteStoreCRUD(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	id, err := store.Save(newRecord("demo"))
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive id, got %d", id)
	}

	found, err := store.Find(id)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if found.App.Name != "demo" || found.Type != domain.TypeK8s {
		t.Errorf("unexpected record: %+v", found)
	}
	if len(found.Files) != 1 || found.Files[0].Path != "deployment.yaml" {
		t.Errorf("files not round-tripped: %+v", found.Files)
	}
	if found.CreatedAt.IsZero() {
		t.Error("created_at not parsed")
	}

	if _, err := store.Save(newRecord("second")); err != nil {
		t.Fatalf("save second: %v", err)
	}
	count, err := store.Count()
	if err != nil || count != 2 {
		t.Fatalf("expected count 2, got %d err %v", count, err)
	}

	list, err := store.List(0, 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 || list[0].AppName != "second" {
		t.Errorf("unexpected list: %+v", list)
	}
	if len(list[0].Kinds) != 1 || list[0].Kinds[0] != "Deployment" {
		t.Errorf("expected kinds projection, got %+v", list[0].Kinds)
	}
	if err := store.Ping(); err != nil {
		t.Errorf("ping: %v", err)
	}
}
