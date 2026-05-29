# restorecheck

Prove that a restic backup can become usable files again.

Current status: first restore pipeline. `restorecheck run` restores selected paths from a restic snapshot into a temporary directory, runs basic file assertions, then cleans up unless `--keep-workdir` is passed.

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
```

`restorecheck run` currently executes these assertion types:

- `exists`
- `not-empty-file`
- `matches-checksum` — verifies restored file SHA-256 against `sha256`

The config parser already recognizes the planned v1 assertion set:

- `exists`
- `not-empty-file`
- `matches-checksum`
- `min-size`
- `non-empty-dir`
- `sqlite-integrity`
- `command`

Unsupported runner assertions currently fail with evidence instead of silently passing.
