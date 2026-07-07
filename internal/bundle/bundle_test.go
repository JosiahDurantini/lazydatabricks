package bundle

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDetectAllSingleBundle(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "databricks.yml"), `
bundle:
  name: my-bundle
targets:
  dev:
    mode: development
  prod:
    mode: production
resources:
  jobs:
    etl:
      name: ETL Job
`)

	configs, err := DetectAll(dir)
	if err != nil {
		t.Fatalf("DetectAll: %v", err)
	}
	if len(configs) != 1 {
		t.Fatalf("got %d configs, want 1", len(configs))
	}
	cfg := configs[0]
	if cfg.Bundle.Name != "my-bundle" {
		t.Errorf("bundle name = %q, want my-bundle", cfg.Bundle.Name)
	}
	if cfg.RootDir != dir {
		t.Errorf("RootDir = %q, want %q", cfg.RootDir, dir)
	}
	if len(cfg.Targets) != 2 {
		t.Errorf("got %d targets, want 2", len(cfg.Targets))
	}
	if cfg.Resources.Jobs["etl"].Name != "ETL Job" {
		t.Errorf("job etl name = %q, want ETL Job", cfg.Resources.Jobs["etl"].Name)
	}
}

func TestParseMergesIncludes(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "databricks.yml"), `
bundle:
  name: with-includes
include:
  - resources/*.yml
resources:
  jobs:
    root_job:
      name: Root Job
`)
	writeFile(t, filepath.Join(dir, "resources", "jobs.yml"), `
resources:
  jobs:
    included_job:
      name: Included Job
`)
	writeFile(t, filepath.Join(dir, "resources", "pipelines.yml"), `
resources:
  pipelines:
    included_pipeline:
      name: Included Pipeline
`)

	configs, err := DetectAll(dir)
	if err != nil {
		t.Fatalf("DetectAll: %v", err)
	}
	cfg := configs[0]
	if len(cfg.Resources.Jobs) != 2 {
		t.Fatalf("got %d jobs, want 2 (root + included): %v", len(cfg.Resources.Jobs), cfg.Resources.Jobs)
	}
	if cfg.Resources.Jobs["included_job"].Name != "Included Job" {
		t.Errorf("included job not merged: %v", cfg.Resources.Jobs)
	}
	if cfg.Resources.Pipelines["included_pipeline"].Name != "Included Pipeline" {
		t.Errorf("included pipeline not merged: %v", cfg.Resources.Pipelines)
	}
}

func TestDetectAllScansSubdirectories(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "beta", "databricks.yml"), "bundle:\n  name: beta-bundle\n")
	writeFile(t, filepath.Join(dir, "alpha", "databricks.yml"), "bundle:\n  name: alpha-bundle\n")
	writeFile(t, filepath.Join(dir, ".hidden", "databricks.yml"), "bundle:\n  name: hidden\n")
	writeFile(t, filepath.Join(dir, "not-a-bundle", "readme.txt"), "nothing here")

	configs, err := DetectAll(dir)
	if err != nil {
		t.Fatalf("DetectAll: %v", err)
	}
	if len(configs) != 2 {
		t.Fatalf("got %d configs, want 2", len(configs))
	}
	// Sorted by bundle name.
	if configs[0].Bundle.Name != "alpha-bundle" || configs[1].Bundle.Name != "beta-bundle" {
		t.Errorf("got order %q, %q; want alpha-bundle, beta-bundle",
			configs[0].Bundle.Name, configs[1].Bundle.Name)
	}
}

func TestDetectAllNoBundles(t *testing.T) {
	dir := t.TempDir()
	if _, err := DetectAll(dir); err == nil {
		t.Fatal("expected error for directory with no bundles, got nil")
	}
}

func TestDetectWalksUp(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "databricks.yml"), "bundle:\n  name: parent-bundle\n")
	nested := filepath.Join(dir, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg, err := Detect(nested)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if cfg.Bundle.Name != "parent-bundle" {
		t.Errorf("bundle name = %q, want parent-bundle", cfg.Bundle.Name)
	}
}
