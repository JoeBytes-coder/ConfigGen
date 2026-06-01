package storage

import (
	"os"
	"testing"
	"time"

	"configgen/internal/domain"
)

func TestSQLiteStore_SaveAndFind(t *testing.T) {
	dbPath := "test_configgen.db"
	defer os.Remove(dbPath)

	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	rec := domain.ConfigRecord{
		Request: domain.ConfigRequest{
			Type:    "compose",
			AppName: "test-app",
			Image:   "nginx",
			Tag:     "latest",
			Port:    80,
		},
		Result: domain.ConfigResult{
			Type:   "compose",
			Config: "version: \"3.8\"\nservices:\n  test-app:\n    image: nginx:latest",
		},
		CreatedAt: time.Now().UTC().Truncate(time.Second),
	}

	id, err := store.Save(rec)
	if err != nil {
		t.Fatalf("failed to save record: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive ID, got %d", id)
	}

	found, err := store.Find(id)
	if err != nil {
		t.Fatalf("failed to find record: %v", err)
	}

	if found.ID != id {
		t.Errorf("expected ID %d, got %d", id, found.ID)
	}
	if found.Request.Type != "compose" {
		t.Errorf("expected compose type, got %s", found.Request.Type)
	}
	if found.Request.AppName != "test-app" {
		t.Errorf("expected app name test-app, got %s", found.Request.AppName)
	}
	if found.Result.Config != rec.Result.Config {
		t.Errorf("config mismatch")
	}
}

func TestSQLiteStore_FindNotFound(t *testing.T) {
	dbPath := "test_notfound.db"
	defer os.Remove(dbPath)

	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	_, err = store.Find(999)
	if err == nil {
		t.Fatal("expected error for non-existent record")
	}
}

func TestSQLiteStore_CreateTableIfNotExists(t *testing.T) {
	dbPath := "test_new.db"
	defer os.Remove(dbPath)

	store1, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store1.Close()

	store2, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to reopen store: %v", err)
	}
	defer store2.Close()

	rec := domain.ConfigRecord{
		Request: domain.ConfigRequest{
			Type:    "dockerfile",
			AppName: "reopen-test",
			Image:   "alpine",
			Tag:     "3.18",
			Port:    80,
		},
		Result: domain.ConfigResult{
			Type:   "dockerfile",
			Config: "FROM alpine:3.18",
		},
		CreatedAt: time.Now().UTC(),
	}

	id, err := store2.Save(rec)
	if err != nil {
		t.Fatalf("failed to save after reopen: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive ID, got %d", id)
	}
}

func TestSQLiteStore_List(t *testing.T) {
	dbPath := "test_list.db"
	defer os.Remove(dbPath)

	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	records, err := store.List(0, 10)
	if err != nil {
		t.Fatalf("failed to list empty store: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}

	for i := 0; i < 3; i++ {
		rec := domain.ConfigRecord{
			Request: domain.ConfigRequest{
				Type:    "compose",
				AppName: "app-" + string(rune('A'+i)),
				Image:   "nginx",
				Tag:     "latest",
				Port:    80,
			},
			Result: domain.ConfigResult{
				Type:   "compose",
				Config: "test-config",
			},
			CreatedAt: time.Now().UTC(),
		}
		store.Save(rec)
	}

	records, err = store.List(0, 10)
	if err != nil {
		t.Fatalf("failed to list records: %v", err)
	}
	if len(records) != 3 {
		t.Errorf("expected 3 records, got %d", len(records))
	}

	records, err = store.List(0, 2)
	if err != nil {
		t.Fatalf("failed to list with limit: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("expected 2 records, got %d", len(records))
	}

	records, err = store.List(2, 2)
	if err != nil {
		t.Fatalf("failed to list with offset: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("expected 1 record past offset, got %d", len(records))
	}
}
