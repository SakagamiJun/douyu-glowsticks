package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveWritesConfigWithOwnerOnlyPermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	cfg := &Config{
		Cookies: []Cookie{
			{Name: "acf_uid", Value: "secret", Domain: "douyu.com", Path: "/"},
		},
		PushKey: "push-key",
	}

	if err := cfg.Save(path); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("config mode = %v, want 0600", got)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "config.json" {
		t.Fatalf("unexpected files after atomic save: %v", entries)
	}
}
