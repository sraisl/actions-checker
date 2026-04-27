package checker

import (
	"errors"
	"testing"

	"action-version-check/internal/parser"
)

func fetch(version string) func(owner, repo string) (string, error) {
	return func(owner, repo string) (string, error) {
		return version, nil
	}
}

func fetchErr(msg string) func(owner, repo string) (string, error) {
	return func(owner, repo string) (string, error) {
		return "", errors.New(msg)
	}
}

func action(ref string) parser.ActionRef {
	return parser.ActionRef{Owner: "actions", Repo: "checkout", Ref: ref, Line: 1, Col: 7}
}

func TestCheck_CommitSHASkipped(t *testing.T) {
	c := NewChecker(CheckerConfig{})
	result := c.Check(action("a81bbbf8298c0fa03ea29cdc473d45769f953675"), fetchErr("should not be called"))
	if result != nil {
		t.Errorf("expected nil for commit SHA, got %+v", result)
	}
}

func TestCheck_UpToDate(t *testing.T) {
	c := NewChecker(CheckerConfig{})
	result := c.Check(action("v4"), fetch("v4"))
	if result != nil {
		t.Errorf("expected nil for up-to-date action, got %+v", result)
	}
}

func TestCheck_UpToDateVerbose(t *testing.T) {
	c := NewChecker(CheckerConfig{Verbose: true})
	result := c.Check(action("v4"), fetch("v4"))
	if result == nil {
		t.Fatal("expected result in verbose mode")
	}
	if result.Type != "info" {
		t.Errorf("expected type info, got %q", result.Type)
	}
	if result.IsError {
		t.Error("up-to-date should not be an error")
	}
}

func TestCheck_Outdated(t *testing.T) {
	c := NewChecker(CheckerConfig{})
	result := c.Check(action("v3"), fetch("v4"))
	if result == nil {
		t.Fatal("expected result for outdated action")
	}
	if result.Type != "warning" {
		t.Errorf("expected type warning, got %q", result.Type)
	}
	if !result.IsError {
		t.Error("outdated action should be an error")
	}
}

func TestCheck_OutdatedSemVer(t *testing.T) {
	c := NewChecker(CheckerConfig{})
	result := c.Check(action("v3.1.0"), fetch("v3.2.0"))
	if result == nil {
		t.Fatal("expected result for outdated semver")
	}
	if !result.IsError {
		t.Error("outdated semver should be IsError")
	}
}

func TestCheck_UnpinnedMainBranch(t *testing.T) {
	c := NewChecker(CheckerConfig{})
	result := c.Check(action("main"), fetchErr("should not be called"))
	if result == nil {
		t.Fatal("expected warning for main branch")
	}
	if result.Type != "warning" {
		t.Errorf("expected warning, got %q", result.Type)
	}
	if result.IsError {
		t.Error("branch warning should not be IsError")
	}
}

func TestCheck_UnpinnedMasterBranch(t *testing.T) {
	c := NewChecker(CheckerConfig{})
	result := c.Check(action("master"), fetchErr("should not be called"))
	if result == nil {
		t.Fatal("expected warning for master branch")
	}
}

func TestCheck_UnpinnedLatest(t *testing.T) {
	c := NewChecker(CheckerConfig{})
	result := c.Check(action("latest"), fetchErr("should not be called"))
	if result == nil {
		t.Fatal("expected warning for latest")
	}
}

func TestCheck_FetchError(t *testing.T) {
	c := NewChecker(CheckerConfig{})
	result := c.Check(action("v4"), fetchErr("connection refused"))
	if result == nil {
		t.Fatal("expected error result when fetch fails")
	}
	if result.Type != "error" {
		t.Errorf("expected type error, got %q", result.Type)
	}
	if !result.IsError {
		t.Error("fetch error should be IsError")
	}
}

func TestCompareVersions_MajorUpgrade(t *testing.T) {
	if compareVersions("v3", "v4") >= 0 {
		t.Error("v3 should be less than v4")
	}
}

func TestCompareVersions_MinorUpgrade(t *testing.T) {
	if compareVersions("3.1.0", "3.2.0") >= 0 {
		t.Error("3.1.0 should be less than 3.2.0")
	}
}

func TestCompareVersions_PatchUpgrade(t *testing.T) {
	if compareVersions("3.1.0", "3.1.1") >= 0 {
		t.Error("3.1.0 should be less than 3.1.1")
	}
}

func TestCompareVersions_Equal(t *testing.T) {
	if compareVersions("v4.2.0", "v4.2.0") != 0 {
		t.Error("equal versions should return 0")
	}
}

func TestCompareVersions_Newer(t *testing.T) {
	if compareVersions("v5", "v4") <= 0 {
		t.Error("v5 should be greater than v4")
	}
}
