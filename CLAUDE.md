# action-version-check

Dieses Projekt ist ein Go-CLI-Tool, das GitHub Actions Workflow-Dateien auf veraltete
Action-Versionen prüft. Es ist als eigenständiges Binary konzipiert, das sich sowohl
als JetBrains File Watcher als auch als Pre-commit Hook einsetzen lässt.

## Ziel

Beim Schreiben von Workflows offline sofort sehen, ob ein `uses: owner/repo@vX.Y.Z`
veraltet ist — als Inline-Annotation direkt im Editor, ohne Plugin-Entwicklung.

## Architektur

```
cmd/
  action-version-check/
    main.go          # CLI entrypoint, Argument-Parsing, Exit-Codes
internal/
  parser/
    parser.go        # YAML parsen, uses:-Zeilen extrahieren
  resolver/
    resolver.go      # GitHub API abfragen, Cache lesen/schreiben
  checker/
    checker.go       # Versionen vergleichen, Errors bauen
go.mod
go.sum
CLAUDE.md
```

## Kernlogik

### 1. Parser (`internal/parser`)

Liest eine YAML-Datei und extrahiert alle `uses:`-Vorkommen mit ihrer Zeilennummer.

- Regex: `uses:\s+([a-zA-Z0-9_.-]+\/[a-zA-Z0-9_.-]+)@([^\s#\n]+)`
- Erfasst: `owner`, `repo`, `ref` (z.B. `v3`, `v3.1.0`, Commit-SHA)
- Ignoriert: lokale Actions (`uses: ./...`), `docker://`-Actions
- Gibt zurück: `[]ActionRef{Owner, Repo, Ref, Line, Col}`

### 2. Resolver (`internal/resolver`)

Holt die neueste Version einer Action von der GitHub API.

**Endpoint:**
```
GET https://api.github.com/repos/{owner}/{repo}/releases/latest
```

Für Actions ohne Releases (z.B. nur Tags):
```
GET https://api.github.com/repos/{owner}/{repo}/tags
```

**Cache:**
- Pfad: `~/.cache/action-version-check/{owner}-{repo}.json`
- Struktur: `{"latest": "v4.2.0", "fetched_at": "2026-04-27T10:00:00Z"}`
- TTL: 6 Stunden (konfigurierbar via `--cache-ttl`)
- Bei fehlendem Netz und vorhandenem Cache: Cache verwenden, keine Warnung

**GitHub Token:**
- Aus Env-Variable `GITHUB_TOKEN` oder `GH_TOKEN`
- Ohne Token: API-Rate-Limit 60 req/h (reicht für normalen Betrieb)
- Mit Token: 5000 req/h

**GHES-Support:**
- Env-Variable `GITHUB_API_URL` überschreibt den API-Endpunkt
- Default: `https://api.github.com`

### 3. Checker (`internal/checker`)

Vergleicht `ref` aus dem Workflow mit der neuesten Version.

- Wenn `ref` eine Commit-SHA ist (40 Hex-Zeichen): überspringen (bewusst gepinnt)
- Wenn `ref == latest`: kein Fehler
- Wenn `ref` ein Major-Tag ist (`v3`): prüfen ob neuere Major-Version existiert
- Wenn `ref` ein SemVer-Tag ist (`v3.1.0`): prüfen ob neuere Version existiert
- Wenn `ref` ein Branch-Name ist (`main`, `master`): Warnung ausgeben (unpinned)

## Output-Format

Das Tool gibt Ergebnisse im Format aus, das JetBrains File Watcher direkt parsen kann:

```
{filepath}:{line}:{col}: {message}
```

Beispiele:
```
/home/stefan/project/.github/workflows/ci.yml:12:11: actions/checkout@v3 is outdated, latest is v4.2.0
/home/stefan/project/.github/workflows/ci.yml:18:11: actions/setup-go@v4 is up to date
/home/stefan/project/.github/workflows/ci.yml:25:11: actions/cache@main is unpinned (branch ref)
```

Nur Warnings/Errors werden ausgegeben (keine up-to-date Meldungen), außer bei `--verbose`.

**Exit-Codes:**
- `0`: Alles up to date
- `1`: Mindestens eine veraltete Action gefunden
- `2`: Fehler (Datei nicht lesbar, API nicht erreichbar ohne Cache)

## CLI-Interface

```
action-version-check [flags] <file|directory> [<file|directory>...]

Flags:
  --cache-ttl duration    Cache-TTL (default: 6h)
  --cache-dir string      Cache-Verzeichnis (default: ~/.cache/action-version-check)
  --github-api-url string GitHub API URL (default: https://api.github.com)
  --format string         Output-Format: jetbrains|github|text (default: jetbrains)
  --verbose               Auch up-to-date Actions ausgeben
  --offline               Nur Cache verwenden, keine API-Calls
  --no-cache              Cache deaktivieren
  -h, --help
```

Mit `--format github` wird das GitHub Actions Annotations-Format ausgegeben:
```
::warning file={path},line={line},col={col}::{message}
```

Damit lässt sich das Tool auch direkt in einem GitHub Actions Workflow nutzen.

## JetBrains File Watcher Konfiguration

```
Name:            action-version-check
Program:         /usr/local/bin/action-version-check
Arguments:       $FilePath$
Output filters:  $FILE_PATH$:$LINE$:$COL$:$MESSAGE$
Working dir:     $ProjectFileDir$
Scope:           Files: .github/workflows//*.yml||.github/workflows//*.yaml
```

Unter `Settings → Editor → Inspections` muss **File Watcher Problems** aktiviert sein.

## Pre-commit Hook

`.pre-commit-config.yaml`:
```yaml
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

## Dependencies

So wenig wie möglich. Angestrebte Abhängigkeiten:

- `gopkg.in/yaml.v3` — YAML-Parsing (oder `go-yaml/yaml`)
- Keine weiteren externen Abhängigkeiten; HTTP-Client und Cache mit stdlib

## Build & Install

```bash
go build -o action-version-check ./cmd/action-version-check
go install github.com/stefanXXX/action-version-check/cmd/action-version-check@latest
```

## Nicht in Scope

- Kein automatisches Update der Workflow-Dateien (nur Reporting)
- Kein Support für Actions die ausschließlich über Docker Hub veröffentlicht werden
- Kein LSP / Language Server
- Keine Prüfung von `action.yml`-Inputs oder -Outputs (das ist actionlint's Job)
