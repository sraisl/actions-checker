package checker

import (
	"regexp"
	"strings"

	"action-version-check/internal/parser"
)

type CheckerConfig struct {
	Verbose bool
}

type Checker struct {
	config CheckerConfig
}

type Result struct {
	Type    string
	Line    int
	Col     int
	Message string
	IsError bool
}

func NewChecker(config CheckerConfig) *Checker {
	return &Checker{config: config}
}

func (c *Checker) Check(action parser.ActionRef, fetchLatest func(owner, repo string) (string, error)) *Result {
	shaRegex := regexp.MustCompile(`^[a-fA-F0-9]{40}$`)

	if shaRegex.MatchString(action.Ref) {
		return nil
	}

	if action.Ref == "latest" || action.Ref == "master" || action.Ref == "main" {
		return &Result{
			Type:    "warning",
			Line:    action.Line,
			Col:     action.Col,
			Message: action.Owner + "/" + action.Repo + "@" + action.Ref + " is unpinned (branch ref)",
			IsError: false,
		}
	}

	latest, err := fetchLatest(action.Owner, action.Repo)
	if err != nil {
		return &Result{
			Type:    "error",
			Line:    action.Line,
			Col:     action.Col,
			Message: "Konnte neueste Version nicht abrufen: " + err.Error(),
			IsError: true,
		}
	}

	comp := compareVersions(action.Ref, latest)

	if comp == 0 {
		if c.config.Verbose {
			return &Result{
				Type:    "info",
				Line:    action.Line,
				Col:     action.Col,
				Message: action.Owner + "/" + action.Repo + "@" + action.Ref + " is up to date",
				IsError: false,
			}
		}
		return nil
	}

	if comp < 0 {
		return &Result{
			Type:    "warning",
			Line:    action.Line,
			Col:     action.Col,
			Message: action.Owner + "/" + action.Repo + "@" + action.Ref + " is outdated, latest is " + latest,
			IsError: true,
		}
	}

	return nil
}

func compareVersions(current, latest string) int {
	current = strings.TrimPrefix(current, "v")
	latest = strings.TrimPrefix(latest, "v")

	cSem := parseSemver(current)
	lSem := parseSemver(latest)

	if cSem.major != lSem.major {
		if cSem.major < lSem.major {
			return -1
		}
		return 1
	}

	if cSem.major == 0 && current != latest {
		if current < latest {
			return -1
		}
		return 1
	}

	if cSem.minor != lSem.minor {
		if cSem.minor < lSem.minor {
			return -1
		}
		return 1
	}

	if cSem.patch != lSem.patch {
		if cSem.patch < lSem.patch {
			return -1
		}
		return 1
	}

	return 0
}

type semver struct {
	major int
	minor int
	patch int
}

func parseSemver(v string) semver {
	parts := strings.Split(v, ".")
	s := semver{}
	if len(parts) > 0 {
		s.major = atoi(parts[0])
	}
	if len(parts) > 1 {
		s.minor = atoi(parts[1])
	}
	if len(parts) > 2 {
		s.patch = atoi(parts[2])
	}
	return s
}

func atoi(s string) int {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			break
		}
	}
	return n
}