package resolver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTestResolver(t *testing.T, serverURL string) *Resolver {
	t.Helper()
	return NewResolver(ResolverConfig{
		APIBaseURL: serverURL,
		CacheDir:   t.TempDir(),
		CacheTTL:   6 * time.Hour,
	})
}

func TestGetLatestVersion_FromRelease(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/actions/checkout/releases/latest" {
			json.NewEncoder(w).Encode(map[string]string{"tag_name": "v4.2.0"})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	r := newTestResolver(t, srv.URL)
	version, err := r.GetLatestVersion("actions", "checkout")
	if err != nil {
		t.Fatal(err)
	}
	if version != "4.2.0" {
		t.Errorf("expected 4.2.0, got %q", version)
	}
}

func TestGetLatestVersion_FallbackToTags(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/actions/checkout/releases/latest":
			w.WriteHeader(http.StatusNotFound)
		case "/repos/actions/checkout/tags":
			json.NewEncoder(w).Encode([]map[string]string{{"name": "v4.1.0"}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	r := newTestResolver(t, srv.URL)
	version, err := r.GetLatestVersion("actions", "checkout")
	if err != nil {
		t.Fatal(err)
	}
	if version != "4.1.0" {
		t.Errorf("expected 4.1.0, got %q", version)
	}
}

func TestGetLatestVersion_CacheHit(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		json.NewEncoder(w).Encode(map[string]string{"tag_name": "v4.2.0"})
	}))
	defer srv.Close()

	r := newTestResolver(t, srv.URL)

	// First call — populates cache
	if _, err := r.GetLatestVersion("actions", "checkout"); err != nil {
		t.Fatal(err)
	}
	// Second call — should use cache
	if _, err := r.GetLatestVersion("actions", "checkout"); err != nil {
		t.Fatal(err)
	}

	if callCount != 1 {
		t.Errorf("expected 1 API call due to cache, got %d", callCount)
	}
}

func TestGetLatestVersion_ExpiredCache(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		json.NewEncoder(w).Encode(map[string]string{"tag_name": "v4.2.0"})
	}))
	defer srv.Close()

	cacheDir := t.TempDir()
	r := NewResolver(ResolverConfig{
		APIBaseURL: srv.URL,
		CacheDir:   cacheDir,
		CacheTTL:   1 * time.Millisecond,
	})

	if _, err := r.GetLatestVersion("actions", "checkout"); err != nil {
		t.Fatal(err)
	}

	time.Sleep(5 * time.Millisecond)

	if _, err := r.GetLatestVersion("actions", "checkout"); err != nil {
		t.Fatal(err)
	}

	if callCount != 2 {
		t.Errorf("expected 2 API calls after cache expiry, got %d", callCount)
	}
}

func TestGetLatestVersion_NoCache(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		json.NewEncoder(w).Encode(map[string]string{"tag_name": "v4.2.0"})
	}))
	defer srv.Close()

	r := NewResolver(ResolverConfig{
		APIBaseURL: srv.URL,
		CacheDir:   t.TempDir(),
		CacheTTL:   6 * time.Hour,
		NoCache:    true,
	})

	r.GetLatestVersion("actions", "checkout")
	r.GetLatestVersion("actions", "checkout")

	if callCount != 2 {
		t.Errorf("expected 2 API calls with NoCache=true, got %d", callCount)
	}
}

func TestGetLatestVersion_OfflineWithCache(t *testing.T) {
	cacheDir := t.TempDir()

	entry := CacheEntry{
		Latest:    "4.1.0",
		FetchedAt: time.Now(),
	}
	data, _ := json.Marshal(entry)
	os.WriteFile(filepath.Join(cacheDir, "actions-checkout.json"), data, 0644)

	r := NewResolver(ResolverConfig{
		APIBaseURL: "http://127.0.0.1:0", // unreachable
		CacheDir:   cacheDir,
		CacheTTL:   6 * time.Hour,
		Offline:    true,
	})

	version, err := r.GetLatestVersion("actions", "checkout")
	if err != nil {
		t.Fatal(err)
	}
	if version != "4.1.0" {
		t.Errorf("expected 4.1.0 from cache, got %q", version)
	}
}

func TestGetLatestVersion_OfflineNoCache(t *testing.T) {
	r := NewResolver(ResolverConfig{
		APIBaseURL: "http://127.0.0.1:0",
		CacheDir:   t.TempDir(),
		CacheTTL:   6 * time.Hour,
		Offline:    true,
	})

	_, err := r.GetLatestVersion("actions", "checkout")
	if err == nil {
		t.Error("expected error in offline mode without cache")
	}
}

func TestGetLatestVersion_NetworkErrorNoCache(t *testing.T) {
	r := NewResolver(ResolverConfig{
		APIBaseURL: "http://127.0.0.1:1", // unreachable
		CacheDir:   t.TempDir(),
		CacheTTL:   6 * time.Hour,
	})

	_, err := r.GetLatestVersion("actions", "checkout")
	if err == nil {
		t.Error("expected error when network is unreachable and no cache available")
	}
}
