package validation

import (
	"fmt"
	"strings"

	zeroreg "github.com/sodiqscript111/zeroreg-go"
)

var githubRepoURLPattern = zeroreg.StartOfLine().
	ThenStr("https://github.com/").
	Then(githubPathSegment()).
	ThenStr("/").
	Then(githubPathSegment()).
	Then(zeroreg.Group(zeroreg.Literal(".git")).Optional()).
	Then(zeroreg.Literal("/").Optional()).
	Then(zeroreg.EndOfLine())

func ValidateGitHubRepoURL(repoURL string) error {
	if !githubRepoURLPattern.Test(strings.TrimSpace(repoURL)) {
		return fmt.Errorf("repo_url must be an https GitHub repository URL like https://github.com/owner/repo")
	}

	return nil
}

func NormalizeGitHubRepoURL(repoURL string) string {
	normalized := strings.TrimSuffix(strings.TrimSpace(repoURL), "/")
	return strings.TrimSuffix(normalized, ".git")
}

func githubPathSegment() *zeroreg.Pattern {
	return zeroreg.CharIn("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789._-").OneOrMore()
}
