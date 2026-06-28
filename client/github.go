package client

import (
	"context"
	"unbx/models"
	"unbx/utils"
)

func GetPRFiles(ctx context.Context, githubToken string, repoSlug string, prNumber string) ([]models.PRFileDiff, error) {
	c := utils.NewGithubClient(githubToken, repoSlug)
	return c.GetPRFiles(ctx, prNumber)
}

// PrSuggest posts a batch review to the PR.
// Returns true if violations were posted (caller should exit 1).
func PrSuggest(ctx context.Context, requestBody models.GitHubReviewRequest, githubToken string, repoSlug string, violationCount int, prNumber string) bool {
	c := utils.NewGithubClient(githubToken, repoSlug)
	return c.PrSuggest(ctx, prNumber, requestBody, violationCount)
}
