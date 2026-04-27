package parser

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

var usesRegex = regexp.MustCompile(`uses:\s+([a-zA-Z0-9_.-]+)/([a-zA-Z0-9_.-]+)@([^\s#\n]+)`)

type ActionRef struct {
	Owner string
	Repo  string
	Ref   string
	Line  int
	Col   int
}

func ParseFile(path string) ([]ActionRef, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var actions []ActionRef
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		matches := usesRegex.FindStringSubmatchIndex(line)
		if matches == nil {
			continue
		}

		owner := line[matches[2]:matches[3]]
		repo := line[matches[4]:matches[5]]
		ref := line[matches[6]:matches[7]]

		if ref == "./" || strings.HasPrefix(ref, "./") {
			continue
		}
		if strings.HasPrefix(ref, "docker://") {
			continue
		}

		col := matches[0] + 1

		actions = append(actions, ActionRef{
			Owner: owner,
			Repo:  repo,
			Ref:   ref,
			Line:  lineNum,
			Col:   col,
		})
	}

	return actions, scanner.Err()
}