# restorecheck

Prove that a restic backup can become usable files again.

Current status: foundation only. `restorecheck init` generates a starter config, and `restorecheck check-config` validates that restore drill description. Restore execution comes next.

## Quick start

```bash
restorecheck init
$EDITOR restorecheck.yml
restorecheck check-config --config restorecheck.yml
```

## Sample config

```yaml
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
  keep_on_failure: false

assertions:
  - name: sqlite database restores and opens
    type: sqlite-integrity
    path: /home/app/data/app.db

  - name: site index exists
    type: exists
    path: /home/app/site/public/index.html

  - name: uploads directory is not empty
    type: non-empty-dir
    path: /home/app/data/uploads

  - name: config file is not empty
    type: min-size
    path: /home/app/data/config.json
    bytes: 32

  - name: custom app-specific check
    type: command
    command: ./scripts/validate-restore.sh "$RESTORE_ROOT"
```

Supported assertion types in the config parser:

- `exists`
- `not-empty-file`
- `min-size`
- `non-empty-dir`
- `sqlite-integrity`
- `command`
