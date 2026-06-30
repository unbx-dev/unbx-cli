package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
	"unbx/models"
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
// Retries up to 3 times on secondary rate limit (403), honoring Retry-After header.
func (c *GithubClient) PrSuggest(ctx context.Context, prNumber string, payload models.GitHubReviewRequest, violationCount int) (hasViolations bool) {
	reqBytes, _ := json.Marshal(payload)
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/pulls/%s/reviews", c.repoSlug, prNumber)

	const maxRetries = 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(reqBytes))
		if err != nil {
			log.Fatalf("❌ Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "token "+c.githubToken)
		req.Header.Set("Accept", "application/vnd.github.comfort-fade-preview+json")
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			log.Fatalf("❌ Failed to send batch review to GitHub: %v", err)
		}

		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
			resp.Body.Close()
			fmt.Printf("✨ Success! Posted %d fix suggestion(s) to the PR.\n", violationCount)
			return true
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusForbidden && attempt < maxRetries {
			wait := 60 * time.Second
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if secs, err := strconv.Atoi(ra); err == nil && secs > 0 {
					wait = time.Duration(secs) * time.Second
				}
			}
			log.Printf("⚠️  Secondary rate limit hit (attempt %d/%d). Retrying in %v...", attempt, maxRetries, wait)
			time.Sleep(wait)
			continue
		}

		log.Fatalf("❌ GitHub API returned an error for the batch review. Status: %d\nResponse: %s", resp.StatusCode, string(body))
	}

	log.Fatalf("❌ Exhausted %d retries due to GitHub secondary rate limit.", maxRetries)
	return false
}
