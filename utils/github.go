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
	"strings"
	"time"
	"unbx/models"
)

// ParseDiffValidLines returns the set of new-file line numbers present in the
// unified diff patch (both context lines and added lines). GitHub's review API
// rejects comments on lines that do not appear in any diff hunk.
func ParseDiffValidLines(patch string) map[int]bool {
	valid := make(map[int]bool)
	newLine := 0
	for _, line := range strings.Split(patch, "\n") {
		if strings.HasPrefix(line, "@@ ") {
			// Parse "+new_start[,count]" from "@@ -old +new @@ ..."
			newLine = 0
			for _, field := range strings.Fields(line) {
				if strings.HasPrefix(field, "+") {
					ns := strings.TrimPrefix(field, "+")
					if idx := strings.Index(ns, ","); idx >= 0 {
						ns = ns[:idx]
					}
					if n, err := strconv.Atoi(ns); err == nil {
						newLine = n
					}
					break
				}
			}
			continue
		}
		if newLine == 0 {
			continue
		}
		switch {
		case strings.HasPrefix(line, "-"):
			// removed line: only in old file, don't advance new-file counter
		case strings.HasPrefix(line, "+"), strings.HasPrefix(line, " "):
			valid[newLine] = true
			newLine++
		}
	}
	return valid
}

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
	req.Header.Set("Accept", "application/vnd.github.comfort-fade-preview+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Fatalf("❌ Failed to send batch review to GitHub: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		fmt.Printf("✨ Success! Posted %d fix suggestion(s) to the PR in a single request.\n", violationCount)
	} else {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("❌ GitHub API returned an error for the batch review. Status: %d\nResponse: %s", resp.StatusCode, string(body))
	}

	return true
}
