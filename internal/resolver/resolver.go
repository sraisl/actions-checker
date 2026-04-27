package resolver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ResolverConfig struct {
	APIBaseURL string
	CacheDir  string
	CacheTTL  time.Duration
	NoCache   bool
	Offline   bool
}

type Resolver struct {
	config  ResolverConfig
	client  *http.Client
	version string
	err     error
}

type CacheEntry struct {
	Latest   string    `json:"latest"`
	FetchedAt time.Time `json:"fetched_at"`
}

func NewResolver(config ResolverConfig) *Resolver {
	return &Resolver{
		config: config,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (r *Resolver) GetLatestVersion(owner, repo string) (string, error) {
	if !r.config.NoCache {
		if cached := r.readCache(owner, repo); cached != "" {
			return cached, nil
		}
	}

	if r.config.Offline {
		return "", fmt.Errorf("offline mode: no cache available for %s/%s", owner, repo)
	}

	version, err := r.fetchFromAPI(owner, repo)
	if err != nil {
		if !r.config.NoCache {
			if cached := r.readCache(owner, repo); cached != "" {
				return cached, nil
			}
		}
		return "", err
	}

	if !r.config.NoCache {
		r.writeCache(owner, repo, version)
	}

	return version, nil
}

func (r *Resolver) fetchFromAPI(owner, repo string) (string, error) {
	url := r.config.APIBaseURL + "/repos/" + owner + "/" + repo + "/releases/latest"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	} else if token := os.Getenv("GH_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return r.fetchTags(owner, repo)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	return strings.TrimPrefix(release.TagName, "v"), nil
}

func (r *Resolver) fetchTags(owner, repo string) (string, error) {
	url := r.config.APIBaseURL + "/repos/" + owner + "/" + repo + "/tags"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	} else if token := os.Getenv("GH_TOKEN"); err != nil {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API error: %d", resp.StatusCode)
	}

	var tags []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return "", err
	}

	if len(tags) == 0 {
		return "", fmt.Errorf("no tags found for %s/%s", owner, repo)
	}

	return strings.TrimPrefix(tags[0].Name, "v"), nil
}

func (r *Resolver) cachePath(owner, repo string) string {
	return filepath.Join(r.config.CacheDir, owner+"-"+repo+".json")
}

func (r *Resolver) readCache(owner, repo string) string {
	if r.config.NoCache {
		return ""
	}

	path := r.cachePath(owner, repo)
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return ""
	}

	if time.Since(entry.FetchedAt) > r.config.CacheTTL {
		return ""
	}

	return entry.Latest
}

func (r *Resolver) writeCache(owner, repo, version string) {
	if r.config.NoCache {
		return
	}

	os.MkdirAll(r.config.CacheDir, 0755)

	entry := CacheEntry{
		Latest:   version,
		FetchedAt: time.Now(),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	os.WriteFile(r.cachePath(owner, repo), data, 0644)
}