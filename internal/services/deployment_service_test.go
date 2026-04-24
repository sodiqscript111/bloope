package services

import "testing"

func TestRailpackCacheKeyIsStableForEquivalentRepoURLs(t *testing.T) {
	first := railpackCacheKey("https://github.com/owner/repo")
	second := railpackCacheKey("https://github.com/owner/repo.git/")

	if first != second {
		t.Fatalf("expected equivalent repo URLs to share cache key, got %q and %q", first, second)
	}
}

func TestRailpackCacheKeyChangesForDifferentRepositories(t *testing.T) {
	first := railpackCacheKey("https://github.com/owner/repo")
	second := railpackCacheKey("https://github.com/owner/other")

	if first == second {
		t.Fatalf("expected different repositories to produce different cache keys")
	}
}
