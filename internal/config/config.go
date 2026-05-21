package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

var AllowedAssertionTypes = map[string]bool{
	"exists":           true,
	"not-empty-file":   true,
	"min-size":         true,
	"non-empty-dir":    true,
	"sqlite-integrity": true,
	"command":          true,
}

type Config struct {
	Backend    string      `yaml:"backend"`
	Restic     Restic      `yaml:"restic"`
	Temp       Temp        `yaml:"temp"`
	Assertions []Assertion `yaml:"assertions"`
}

type Restic struct {
	Repository   string   `yaml:"repository"`
	PasswordFile string   `yaml:"password_file"`
	Snapshot     string   `yaml:"snapshot"`
	Paths        []string `yaml:"paths"`
}

type Temp struct {
	Parent        string `yaml:"parent"`
	KeepOnFailure bool   `yaml:"keep_on_failure"`
}

type Assertion struct {
	Name    string `yaml:"name"`
	Type    string `yaml:"type"`
	Path    string `yaml:"path"`
	Bytes   int64  `yaml:"bytes"`
	Command string `yaml:"command"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	return Parse(b)
}

func Parse(b []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	applyDefaults(&cfg)
	if err := Validate(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.Backend == "" {
		cfg.Backend = "restic"
	}
	if cfg.Restic.Snapshot == "" {
		cfg.Restic.Snapshot = "latest"
	}
	if cfg.Temp.Parent == "" {
		cfg.Temp.Parent = os.TempDir()
	}
}

func Validate(cfg *Config) error {
	var problems []string
	if cfg.Backend != "restic" {
		problems = append(problems, "backend must be 'restic'")
	}
	if strings.TrimSpace(cfg.Restic.Repository) == "" && os.Getenv("RESTIC_REPOSITORY") == "" {
		problems = append(problems, "restic.repository is required unless RESTIC_REPOSITORY is set")
	}
	if strings.TrimSpace(cfg.Restic.PasswordFile) == "" && os.Getenv("RESTIC_PASSWORD_FILE") == "" && os.Getenv("RESTIC_PASSWORD") == "" {
		problems = append(problems, "restic.password_file is required unless RESTIC_PASSWORD_FILE or RESTIC_PASSWORD is set")
	}
	if len(cfg.Restic.Paths) == 0 {
		problems = append(problems, "restic.paths must contain at least one path")
	}
	if len(cfg.Assertions) == 0 {
		problems = append(problems, "assertions must contain at least one assertion")
	}
	for i, a := range cfg.Assertions {
		prefix := fmt.Sprintf("assertions[%d]", i)
		if strings.TrimSpace(a.Name) == "" {
			problems = append(problems, prefix+".name is required")
		}
		if !AllowedAssertionTypes[a.Type] {
			problems = append(problems, fmt.Sprintf("%s.type %q is not supported", prefix, a.Type))
			continue
		}
		switch a.Type {
		case "exists", "not-empty-file", "non-empty-dir", "sqlite-integrity":
			if strings.TrimSpace(a.Path) == "" {
				problems = append(problems, prefix+".path is required for type "+a.Type)
			}
		case "min-size":
			if strings.TrimSpace(a.Path) == "" {
				problems = append(problems, prefix+".path is required for type min-size")
			}
			if a.Bytes <= 0 {
				problems = append(problems, prefix+".bytes must be greater than 0 for type min-size")
			}
		case "command":
			if strings.TrimSpace(a.Command) == "" {
				problems = append(problems, prefix+".command is required for type command")
			}
		}
	}
	if len(problems) > 0 {
		return errors.New("invalid config:\n- " + strings.Join(problems, "\n- "))
	}
	return nil
}

const StarterConfig = `# restorecheck.yml
# Describe one restore drill. restorecheck will restore the configured restic
# snapshot into a temporary directory, run these assertions, then clean up.
backend: restic

restic:
  repository: /srv/backups/restic
  password_file: /etc/restic/password
  snapshot: latest
  paths:
    - /home/app/data
    - /home/app/site

temp:
  parent: /tmp

assertions:
  - name: site index exists
    type: exists
    path: /home/app/site/public/index.html

  - name: config file is not empty
    type: not-empty-file
    path: /home/app/data/config.json
`
