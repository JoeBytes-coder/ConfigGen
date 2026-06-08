package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetListenAddr(t *testing.T) {
	origAddr := os.Getenv("ADDR")
	origPort := os.Getenv("PORT")
	defer func() {
		if origAddr == "" {
			os.Unsetenv("ADDR")
		} else {
			os.Setenv("ADDR", origAddr)
		}
		if origPort == "" {
			os.Unsetenv("PORT")
		} else {
			os.Setenv("PORT", origPort)
		}
	}()

	os.Unsetenv("ADDR")
	os.Unsetenv("PORT")

	if got := GetListenAddr(""); got != ":8080" {
		t.Fatalf("expected default :8080, got %q", got)
	}

	os.Setenv("PORT", "3001")
	if got := GetListenAddr(""); got != ":3001" {
		t.Fatalf("expected :3001 from PORT, got %q", got)
	}

	os.Setenv("ADDR", ":9000")
	if got := GetListenAddr(""); got != ":9000" {
		t.Fatalf("expected :9000 from ADDR, got %q", got)
	}

	if got := GetListenAddr(":1234"); got != ":1234" {
		t.Fatalf("expected explicit addr :1234, got %q", got)
	}
}

func TestLoadDotEnv(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, ".env")
	content := `# Example dotenv file
PORT=4000
ADDR=:5000
CONFIGGEN_DB=db/test.db
FOO="bar baz"
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	orig := map[string]string{
		"PORT":         os.Getenv("PORT"),
		"ADDR":         os.Getenv("ADDR"),
		"CONFIGGEN_DB": os.Getenv("CONFIGGEN_DB"),
	}
	defer func() {
		for k, v := range orig {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	os.Unsetenv("PORT")
	os.Unsetenv("ADDR")
	os.Unsetenv("CONFIGGEN_DB")

	if err := LoadDotEnv(path); err != nil {
		t.Fatal(err)
	}

	if got := os.Getenv("PORT"); got != "4000" {
		t.Fatalf("expected PORT=4000, got %q", got)
	}
	if got := os.Getenv("ADDR"); got != ":5000" {
		t.Fatalf("expected ADDR=:5000, got %q", got)
	}
	if got := os.Getenv("CONFIGGEN_DB"); got != "db/test.db" {
		t.Fatalf("expected CONFIGGEN_DB=db/test.db, got %q", got)
	}
}
