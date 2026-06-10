# restorecheck

Prove that a restic backup can become usable files again.

`restorecheck run` restores selected paths from a restic snapshot into a temporary directory, runs file, directory, checksum, and custom command assertions, reports evidence, then cleans up unless `--keep-workdir` is passed. Backup monitoring should prove restores, not just successful backup commands.

Current status: working restore pipeline with config validation, checksum/min-size/non-empty-dir assertions, custom command assertions, and test coverage for parser + runner behavior.

## Quick start

```bash
restorecheck init
$EDITOR restorecheck.yml
restorecheck check-config --config restorecheck.yml
restorecheck run --config restorecheck.yml
```

## Sample config

```yaml
backend: restic

restic:
  repository: /srv/backups/restic
  password_file: /etc/restic/password
  snapshot: latest
  paths:
    - /home/app/data/config.json
    - /home/app/site/public/index.html

temp:
  parent: /tmp

assertions:
  - name: site index exists
    type: exists
    path: /home/app/site/public/index.html

  - name: config file is not empty
    type: not-empty-file
    path: /home/app/data/config.json

  - name: config file has expected checksum
    type: matches-checksum
    path: /home/app/data/config.json
    sha256: e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855

  - name: uploaded media directory has restored content
    type: non-empty-dir
    path: /home/app/uploads

  - name: sqlite database is at least 1 MiB
    type: min-size
    path: /home/app/data/app.db
    bytes: 1048576

  - name: app-specific restore validator passes
    type: command
    command: ./validate-restore.sh "$RESTORE_ROOT"
```

`restorecheck run` currently executes these assertion types:

- `exists`
- `not-empty-file`
- `matches-checksum` — verifies restored file SHA-256 against `sha256`
- `min-size` — verifies a restored file is at least `bytes` bytes
- `non-empty-dir` — verifies a restored directory contains at least one entry
- `command` — runs a shell command from the restore root with `RESTORE_ROOT` set to the temporary restore directory

The config parser also recognizes the planned `sqlite-integrity` assertion. Unsupported runner assertions currently fail with evidence instead of silently passing.

## Verification

```bash
go test ./...
```

The test suite covers config parsing and runner assertion behavior. For production use, point the sample config at a real restic repository and run `restorecheck run --config restorecheck.yml` from the host that can read the repository/password file.
