package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"unbx/models"
	"log"
	"net/http"
	"time"
)

type GithubClient struct {
	githubToken string
	repoSlug    string
	httpClient  *http.Client
}

func NewGithubClient(githubToken string, repoSlug string) *GithubClient {
	return &GithubClient{
		githubToken: githubToken,
		repoSlug:    repoSlug,
		httpClient:  &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *GithubClient) GetPRFiles(ctx context.Context, prNumber string) ([]models.PRFileDiff, error) {
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/pulls/%s/files", c.repoSlug, prNumber)
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "token "+c.githubToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PR files: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API error: status %d", resp.StatusCode)
	}

	var raw []struct {
		Filename string `json:"filename"`
		Patch    string `json:"patch"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to decode PR files: %w", err)
	}

	files := make([]models.PRFileDiff, 0, len(raw))
	for _, f := range raw {
		files = append(files, models.PRFileDiff{
			Path:      f.Filename,
			PatchCode: f.Patch,
		})
	}
	return files, nil
}

// PrSuggest posts a batch review to the PR and returns whether violations were found.
// Callers are responsible for exiting with a non-zero code when this returns true.
func (c *GithubClient) PrSuggest(ctx context.Context, prNumber string, payload models.GitHubReviewRequest, violationCount int) (hasViolations bool) {
	reqBytes, _ := json.Marshal(payload)

	// POST /repos/{owner}/{repo}/pulls/{pull_number}/reviews
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/pulls/%s/reviews", c.repoSlug, prNumber)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(reqBytes))
	if err != nil {
		log.Fatalf("❌ Failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "token "+c.githubToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Fatalf("❌ Failed to send batch review to GitHub: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		fmt.Printf("✨ Success! Posted %d fix suggestion(s) to the PR in a single request.\n", violationCount)
	} else {
		log.Fatalf("❌ GitHub API returned an error for the batch review. Status: %d", resp.StatusCode)
	}

	return true
}
