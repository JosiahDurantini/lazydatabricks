package bundle

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Bundle struct {
		Name string `yaml:"name"`
	} `yaml:"bundle"`
	Include   []string          `yaml:"include"`
	Targets   map[string]Target `yaml:"targets"`
	Resources Resources         `yaml:"resources"`
	RootDir   string
}

type Target struct {
	Mode string `yaml:"mode"`
	Host string `yaml:"host"`
}

type Resources struct {
	Jobs      map[string]JobDef      `yaml:"jobs"`
	Pipelines map[string]PipelineDef `yaml:"pipelines"`
}

type JobDef struct {
	Name string `yaml:"name"`
}

type PipelineDef struct {
	Name string `yaml:"name"`
}

// partial is used to parse included files which may only have resources.
type partial struct {
	Resources Resources `yaml:"resources"`
}

// Detect walks up from dir until it finds a databricks.yml.
func Detect(dir string) (*Config, error) {
	for {
		path := filepath.Join(dir, "databricks.yml")
		if _, err := os.Stat(path); err == nil {
			return parse(path)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, fmt.Errorf("no databricks.yml found in %s or any parent directory", dir)
		}
		dir = parent
	}
}

// DetectAll returns bundles for the given directory:
//   - If dir contains databricks.yml, returns that single bundle.
//   - Otherwise scans immediate subdirectories for databricks.yml files.
func DetectAll(dir string) ([]*Config, error) {
	if _, err := os.Stat(filepath.Join(dir, "databricks.yml")); err == nil {
		cfg, err := parse(filepath.Join(dir, "databricks.yml"))
		if err != nil {
			return nil, err
		}
		return []*Config{cfg}, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var configs []*Config
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		p := filepath.Join(dir, e.Name(), "databricks.yml")
		if _, err := os.Stat(p); err == nil {
			cfg, err := parse(p)
			if err != nil {
				continue
			}
			configs = append(configs, cfg)
		}
	}

	if len(configs) == 0 {
		return nil, fmt.Errorf("no databricks.yml found in %s or its subdirectories", dir)
	}

	sort.Slice(configs, func(i, j int) bool {
		return configs[i].Bundle.Name < configs[j].Bundle.Name
	})

	return configs, nil
}

func parse(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	cfg.RootDir = filepath.Dir(path)

	// Resolve include globs and merge their resources into the root config.
	for _, pattern := range cfg.Include {
		if !filepath.IsAbs(pattern) {
			pattern = filepath.Join(cfg.RootDir, pattern)
		}
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		for _, match := range matches {
			inc, err := parsePartial(match)
			if err != nil {
				continue
			}
			if cfg.Resources.Jobs == nil {
				cfg.Resources.Jobs = make(map[string]JobDef)
			}
			for k, v := range inc.Resources.Jobs {
				cfg.Resources.Jobs[k] = v
			}
			if cfg.Resources.Pipelines == nil {
				cfg.Resources.Pipelines = make(map[string]PipelineDef)
			}
			for k, v := range inc.Resources.Pipelines {
				cfg.Resources.Pipelines[k] = v
			}
		}
	}

	return &cfg, nil
}

func parsePartial(path string) (*partial, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p partial
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}
