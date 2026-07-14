package config

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadMissingFileReturnsEmptyConfig(t *testing.T) {
	cfg, err := LoadLabels(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if cfg.Version != 1 || len(cfg.Orgs) != 0 {
		t.Fatalf("Load() = %+v, want empty v1 config", cfg)
	}
}

func TestSaveThenLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "labels.yaml")

	cfg := &LabelConfig{Version: 1, Orgs: map[string]OrgLabels{}}
	cfg.SetLabels("my-org", "backend-api", []string{"backend", "critical"})
	cfg.SetLabels("my-org", "frontend-app", []string{"frontend"})

	if err := cfg.Save(path); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := LoadLabels(path)
	if err != nil {
		t.Fatalf("LoadLabels() error = %v", err)
	}

	if got := loaded.LabelsFor("my-org", "backend-api"); !reflect.DeepEqual(got, []string{"backend", "critical"}) {
		t.Errorf("LabelsFor(backend-api) = %v, want [backend critical]", got)
	}
	if got := loaded.LabelsFor("my-org", "unknown-repo"); got != nil {
		t.Errorf("LabelsFor(unknown-repo) = %v, want nil", got)
	}
}

func TestReposByLabel(t *testing.T) {
	cfg := &LabelConfig{Version: 1, Orgs: map[string]OrgLabels{}}
	cfg.SetLabels("my-org", "backend-api", []string{"backend", "critical"})
	cfg.SetLabels("my-org", "worker", []string{"backend"})
	cfg.SetLabels("my-org", "frontend-app", []string{"frontend"})

	got := cfg.ReposByLabel("my-org", "backend")
	want := []string{"backend-api", "worker"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ReposByLabel(backend) = %v, want %v", got, want)
	}

	if got := cfg.ReposByLabel("my-org", "nonexistent"); len(got) != 0 {
		t.Errorf("ReposByLabel(nonexistent) = %v, want empty", got)
	}
}
