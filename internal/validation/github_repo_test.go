package validation

import "testing"

func TestNormalizeGitHubRepoURLCanonicalizesGitSuffix(t *testing.T) {
	got := NormalizeGitHubRepoURL("https://github.com/owner/repo.git/")
	want := "https://github.com/owner/repo"

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
