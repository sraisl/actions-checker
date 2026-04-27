# action-version-check

CLI-Tool zur Prüfung von veralteten GitHub Actions Versionen in Workflow-Dateien.

## Installation

```bash
go install github.com/stefanXXX/action-version-check/cmd/action-version-check@latest
```

Oder lokal bauen:

```bash
go build -o action-version-check ./cmd/action-version-check
mv action-version-check /usr/local/bin/
```

## Verwendung

```bash
action-version-check [flags] <file|directory>...

# Einzelne Datei
action-version-check .github/workflows/ci.yml

# Mehrere Dateien
action-version-check .github/workflows/*.yml

# Ganzes .github Verzeichnis
action-version-check .github/
```

## Flags

| Flag | Default | Beschreibung |
|------|---------|--------------|
| `--cache-ttl` | 6h | Cache-TTL |
| `--cache-dir` | ~/.cache/action-version-check | Cache-Verzeichnis |
| `--github-api-url` | https://api.github.com | GitHub API URL (für GHES) |
| `--format` | jetbrains | Output-Format: jetbrains, github, text |
| `--verbose` | false | Auch up-to-date Actions ausgeben |
| `--offline` | false | Nur Cache verwenden |
| `--no-cache` | false | Cache deaktivieren |
| `-h`, `--help` | | Hilfe anzeigen |

## Output-Formate

### jetbrains (Standard)
```
/path/to/workflow.yml:12:9: actions/checkout@v3 is outdated, latest is v4.0.0
```

### github
```
::warning file=/path/to/workflow.yml,line=12,col=9::actions/checkout@v3 is outdated, latest is v4.0.0
```

### text
```
/path/to/workflow.yml (warning): actions/checkout@v3 is outdated, latest is v4.0.0
```

## Exit-Codes

- `0`: Alle Actions up-to-date
- `1`: Mindestens eine veraltete Action gefunden
- `2`: Fehler (Datei nicht lesbar, API-Fehler)

## IntelliJ File Watcher

1. **Settings → Tools → File Watchers** → `+` hinzufügen
2. Konfiguration:
   ```
   Name:           action-version-check
   Program:        /usr/local/bin/action-version-check
   Arguments:      $FilePath$
   Output filters: $FILE_PATH$:$LINE$:$COL$:$MESSAGE$
   Working dir:    $ProjectFileDir$
   ```
3. **Scope:** `.github/workflows//*.yml`
4. **File Watcher Problems** unter Settings → Editor → Inspections aktivieren

## Pre-commit Hook

```yaml
# .pre-commit-config.yaml
repos:
  - repo: local
    hooks:
      - id: action-version-check
        name: Check GitHub Action versions
        entry: action-version-check
        language: system
        files: ^\.github/workflows/.*\.ya?ml$
        pass_filenames: true
```

```bash
pre-commit install
```

## Umgebungsvariablen

| Variable | Beschreibung |
|----------|--------------|
| `GITHUB_TOKEN` | GitHub API Token (erhöht Rate-Limit auf 5000/h) |
| `GH_TOKEN` | Alias für GITHUB_TOKEN |
| `GITHUB_API_URL` | Override für API-URL (z.B. für GHES) |

## Cache

Cache wird gespeichert unter:
```
~/.cache/action-version-check/{owner}-{repo}.json
```

Format:
```json
{"latest": "v4.2.0", "fetched_at": "2026-04-27T10:00:00Z"}
```

## Lizenz

MIT