package parser

import (
	"os"
	"testing"
)

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "workflow-*.yml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

func TestParseFile_BasicAction(t *testing.T) {
	path := writeTemp(t, `
jobs:
  build:
    steps:
      - uses: actions/checkout@v4
`)
	refs, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d", len(refs))
	}
	ref := refs[0]
	if ref.Owner != "actions" {
		t.Errorf("owner: got %q, want %q", ref.Owner, "actions")
	}
	if ref.Repo != "checkout" {
		t.Errorf("repo: got %q, want %q", ref.Repo, "checkout")
	}
	if ref.Ref != "v4" {
		t.Errorf("ref: got %q, want %q", ref.Ref, "v4")
	}
	if ref.Line != 5 {
		t.Errorf("line: got %d, want %d", ref.Line, 5)
	}
}

func TestParseFile_MultipleActions(t *testing.T) {
	path := writeTemp(t, `
steps:
  - uses: actions/checkout@v4
  - uses: actions/setup-go@v5
  - uses: actions/cache@v3.1.0
`)
	refs, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 3 {
		t.Fatalf("expected 3 refs, got %d", len(refs))
	}
	if refs[0].Repo != "checkout" {
		t.Errorf("refs[0].Repo = %q", refs[0].Repo)
	}
	if refs[1].Repo != "setup-go" {
		t.Errorf("refs[1].Repo = %q", refs[1].Repo)
	}
	if refs[2].Repo != "cache" {
		t.Errorf("refs[2].Repo = %q", refs[2].Repo)
	}
}

func TestParseFile_IgnoresLocalAction(t *testing.T) {
	path := writeTemp(t, `
steps:
  - uses: ./my-local-action
`)
	refs, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 0 {
		t.Errorf("expected 0 refs for local action, got %d", len(refs))
	}
}

func TestParseFile_IgnoresDockerAction(t *testing.T) {
	path := writeTemp(t, `
steps:
  - uses: docker://alpine:3.18
`)
	refs, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 0 {
		t.Errorf("expected 0 refs for docker action, got %d", len(refs))
	}
}

func TestParseFile_IgnoresComment(t *testing.T) {
	path := writeTemp(t, `
steps:
  - uses: actions/checkout@v4 # pinned to v4
`)
	refs, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d", len(refs))
	}
	if refs[0].Ref != "v4" {
		t.Errorf("ref should not include comment, got %q", refs[0].Ref)
	}
}

func TestParseFile_CommitSHA(t *testing.T) {
	path := writeTemp(t, `
steps:
  - uses: actions/checkout@a81bbbf8298c0fa03ea29cdc473d45769f953675
`)
	refs, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d", len(refs))
	}
	if refs[0].Ref != "a81bbbf8298c0fa03ea29cdc473d45769f953675" {
		t.Errorf("unexpected ref: %q", refs[0].Ref)
	}
}

func TestParseFile_NotFound(t *testing.T) {
	_, err := ParseFile("/does/not/exist.yml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}
