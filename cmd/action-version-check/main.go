package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"action-version-check/internal/checker"
	"action-version-check/internal/parser"
	"action-version-check/internal/resolver"
)

var (
	cacheTTL      time.Duration
	cacheDir      string
	githubAPIURL  string
	outputFormat  string
	verbose       bool
	offline       bool
	noCache       bool
	showHelp      bool
)

func init() {
	flag.DurationVar(&cacheTTL, "cache-ttl", 6*time.Hour, "Cache-TTL")
	flag.StringVar(&cacheDir, "cache-dir", "", "Cache-Verzeichnis")
	flag.StringVar(&githubAPIURL, "github-api-url", "https://api.github.com", "GitHub API URL")
	flag.StringVar(&outputFormat, "format", "jetbrains", "Output-Format: jetbrains|github|text")
	flag.BoolVar(&verbose, "verbose", false, "Auch up-to-date Actions ausgeben")
	flag.BoolVar(&offline, "offline", false, "Nur Cache verwenden")
	flag.BoolVar(&noCache, "no-cache", false, "Cache deaktivieren")
	flag.BoolVar(&showHelp, "h", false, "Hilfe anzeigen")
	flag.BoolVar(&showHelp, "help", false, "Hilfe anzeigen")
	flag.Parse()
}

func main() {
	if showHelp {
		flag.Usage()
		os.Exit(0)
	}
	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, "Nutzung: action-version-check [flags] <file|directory>...\n")
		flag.VisitAll(func(f *flag.Flag) {
			fmt.Fprintf(os.Stderr, "  -%s %v\n", f.Name, f.DefValue)
		})
		flag.PrintDefaults()
		os.Exit(2)
	}

	if cacheDir == "" {
		home := os.Getenv("HOME")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		cacheDir = home + "/.cache/action-version-check"
	}

	res := resolver.NewResolver(resolver.ResolverConfig{
		APIBaseURL: githubAPIURL,
		CacheDir:  cacheDir,
		CacheTTL:  cacheTTL,
		NoCache:   noCache,
		Offline:   offline,
	})

	checkr := checker.NewChecker(checker.CheckerConfig{
		Verbose: verbose,
	})

	hasErrors := false

	for _, path := range args {
		entries, err := os.Stat(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Fehler: %s\n", err)
			os.Exit(2)
		}

		var files []string
		if entries.IsDir() {
			files, err = findWorkflowFiles(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Fehler: %s\n", err)
				os.Exit(2)
			}
		} else {
			files = []string{path}
		}

		for _, file := range files {
			actions, err := parser.ParseFile(file)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Fehler beim Parsen von %s: %s\n", file, err)
				os.Exit(2)
			}

			for _, action := range actions {
				result := checkr.Check(action, func(owner, repo string) (string, error) {
					return res.GetLatestVersion(owner, repo)
				})

				if result == nil {
					continue
				}

				if result.IsError {
					hasErrors = true
				}

				printResult(file, result, outputFormat)
			}
		}
	}

	if hasErrors {
		os.Exit(1)
	}
}

func findWorkflowFiles(dir string) ([]string, error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var files []string
	for {
		entries, err := f.ReadDir(100)
		if err != nil {
			break
		}
		for _, e := range entries {
			if e.IsDir() && (e.Name() == "workflows" || e.Name() == ".github") {
				subfiles, _ := findWorkflowFiles(dir + "/" + e.Name())
				files = append(files, subfiles...)
			}
			if !e.IsDir() && (e.Name() == "workflows" || (len(e.Name()) > 5 && e.Name()[len(e.Name())-5:] == ".yaml" || e.Name()[len(e.Name())-4:] == ".yml")) {
				files = append(files, dir+"/"+e.Name())
			}
		}
	}
	return files, nil
}

func printResult(file string, result *checker.Result, format string) {
	switch format {
	case "github":
		fmt.Printf("::%s file=%s,line=%d,col=%d::%s\n", result.Type, file, result.Line, result.Col, result.Message)
	case "text":
		fmt.Printf("%s (%s): %s\n", file, result.Type, result.Message)
	default:
		fmt.Printf("%s:%d:%d: %s\n", file, result.Line, result.Col, result.Message)
	}
}