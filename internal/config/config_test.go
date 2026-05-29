package config

import (
	"strings"
	"testing"
)

const validConfig = `backend: restic
restic:
  repository: /srv/backups/restic
  password_file: /etc/restic/password
  snapshot: latest
  paths:
    - /home/app/data
assertions:
  - name: database is valid
    type: sqlite-integrity
    path: /home/app/data/app.db
  - name: config is large enough
    type: min-size
    path: /home/app/data/config.json
    bytes: 32
  - name: custom validator
    type: command
    command: ./validate.sh "$RESTORE_ROOT"
  - name: config checksum
    type: matches-checksum
    path: /home/app/data/config.json
    sha256: 239f59ed55e737c77147cf55ad0c1b030b6d7ee748a7426952f9b852d5a935e5
`

func TestValidConfigLoads(t *testing.T) {
	cfg, err := Parse([]byte(validConfig))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if cfg.Backend != "restic" {
		t.Fatalf("Backend = %q, want restic", cfg.Backend)
	}
	if got := len(cfg.Assertions); got != 4 {
		t.Fatalf("len(Assertions) = %d, want 4", got)
	}
	if cfg.Restic.Snapshot != "latest" {
		t.Fatalf("Snapshot = %q, want latest", cfg.Restic.Snapshot)
	}
}

func TestMissingRequiredFieldsRejected(t *testing.T) {
	_, err := Parse([]byte(`backend: restic
restic:
  password_file: /etc/restic/password
  paths:
    - /home/app/data
assertions:
  - name: file exists
    type: exists
    path: /home/app/data/file.txt
`))
	if err == nil {
		t.Fatal("Parse returned nil error, want validation error")
	}
	if !strings.Contains(err.Error(), "restic.repository is required") {
		t.Fatalf("error = %q, want missing repository", err)
	}
}

func TestUnknownAssertionTypeRejected(t *testing.T) {
	_, err := Parse([]byte(`backend: restic
restic:
  repository: /srv/backups/restic
  password_file: /etc/restic/password
  paths:
    - /home/app/data
assertions:
  - name: mystery check
    type: magic
    path: /home/app/data/file.txt
`))
	if err == nil {
		t.Fatal("Parse returned nil error, want validation error")
	}
	if !strings.Contains(err.Error(), `assertions[0].type "magic" is not supported`) {
		t.Fatalf("error = %q, want unknown assertion type", err)
	}
}

func TestChecksumAssertionRequiresValidSHA256(t *testing.T) {
	_, err := Parse([]byte(`backend: restic
restic:
  repository: /srv/backups/restic
  password_file: /etc/restic/password
  paths:
    - /home/app/data
assertions:
  - name: bad checksum
    type: matches-checksum
    path: /home/app/data/file.txt
    sha256: not-a-digest
`))
	if err == nil {
		t.Fatal("Parse returned nil error, want validation error")
	}
	if !strings.Contains(err.Error(), "sha256 must be a 64-character hex digest") {
		t.Fatalf("error = %q, want sha256 validation error", err)
	}
}

func TestEmptyAssertionsRejected(t *testing.T) {
	_, err := Parse([]byte(`backend: restic
restic:
  repository: /srv/backups/restic
  password_file: /etc/restic/password
  paths:
    - /home/app/data
assertions: []
`))
	if err == nil {
		t.Fatal("Parse returned nil error, want validation error")
	}
	if !strings.Contains(err.Error(), "assertions must contain at least one assertion") {
		t.Fatalf("error = %q, want empty assertions error", err)
	}
}
