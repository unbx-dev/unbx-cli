package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"unbx/client"
	"unbx/models"
	"unbx/utils"
)

func main() {
	repositoryID := os.Getenv("REPOSITORY_ID")
	accessToken := os.Getenv("ACCESS_TOKEN")
	secretToken := os.Getenv("SECRET_TOKEN")
	commitSHA := os.Getenv("UNBX_COMMIT_SHA")
	githubToken := os.Getenv("GITHUB_TOKEN")
	repoSlug := os.Getenv("REPO_SLUG")
	prNumber := os.Getenv("PR_NUMBER")

	ctx := context.Background()

	if accessToken == "" || secretToken == "" {
		log.Fatal("ACCESS_TOKEN and SECRET_TOKEN are required")
	}

	changeFiles, err := client.GetPRFiles(ctx, githubToken, repoSlug, prNumber)
	if err != nil {
		log.Fatal("Failed to fetch PR files:", err)
	}

	// Strip diff markers so tree-sitter sees valid source code, and build a
	// row→fileLine mapping per file to convert scan results to real line numbers.
	type patchMeta struct {
		cleanCode     string
		validLines    map[int]bool
		rowToFileLine []int
	}
	meta := make(map[string]patchMeta, len(changeFiles))
	for _, f := range changeFiles {
		clean, mapping := utils.StripPatch(f.PatchCode)
		valid := make(map[int]bool, len(mapping))
		for _, n := range mapping {
			valid[n] = true
		}
		meta[f.Path] = patchMeta{cleanCode: clean, validLines: valid, rowToFileLine: mapping}
	}

	scanRequest := models.BulkScanRequest{
		GithubRepositoryID: repositoryID,
	}
	for _, fileDiff := range changeFiles {
		langName := utils.LangNameForFile(fileDiff.Path)
		if langName == "" {
			continue
		}

		encryptedSource, err := utils.EncryptSource([]byte(meta[fileDiff.Path].cleanCode), secretToken)
		if err != nil {
			log.Fatalf("Failed to encrypt %s: %v", fileDiff.Path, err)
		}

		scanRequest.Files = append(scanRequest.Files, models.FilePayload{
			Path:            fileDiff.Path,
			EncryptedSource: encryptedSource,
			LangName:        langName,
		})
	}

	scanResponse, err := client.BulkScan(ctx, scanRequest)
	if err != nil {
		log.Fatal("Scan failed:", err)
	}

	if len(scanResponse.Violations) == 0 {
		fmt.Println("✅ No violations found. All good!")
		os.Exit(0)
	}

	// Convert scan row numbers (relative to clean source) to actual file line numbers.
	for i := range scanResponse.Violations {
		v := &scanResponse.Violations[i]
		m := meta[v.FilePath].rowToFileLine
		v.StartLine = utils.PatchRowToFileLine(v.StartLine, m)
		v.EndLine = utils.PatchRowToFileLine(v.EndLine, m)
	}

	validDiffLines := make(map[string]map[int]bool, len(changeFiles))
	for path, m := range meta {
		validDiffLines[path] = m.validLines
	}

	comments := make([]models.GitHubDraftComment, 0, len(scanResponse.Violations))
	for _, violation := range scanResponse.Violations {
		fileLines, ok := validDiffLines[violation.FilePath]
		if !ok || !fileLines[violation.EndLine] {
			// EndLine is not in the diff — skip to avoid 422
			continue
		}
		var commentBody string
		if violation.SuggestedFix != "" {
			commentBody = fmt.Sprintf(
				"### 🚨 Unbx Quarantine Alert: [%s]\n%s\n\n```suggestion\n%s\n```",
				violation.RuleTitle,
				violation.Message,
				violation.SuggestedFix,
			)
		} else {
			commentBody = fmt.Sprintf(
				"### 🚫 Unbx Quarantine Alert: [%s]\n%s",
				violation.RuleTitle,
				violation.Message,
			)
		}
		comment := models.GitHubDraftComment{
			Path: violation.FilePath,
			Body: commentBody,
			Line: violation.EndLine,
			Side: "RIGHT",
		}
		if violation.StartLine > 0 && violation.StartLine < violation.EndLine && fileLines[violation.StartLine] {
			comment.StartLine = violation.StartLine
			comment.StartSide = "RIGHT"
		}
		comments = append(comments, comment)
	}

	const chunkSize = 10
	totalViolations := len(scanResponse.Violations)
	for i := 0; i < len(comments); i += chunkSize {
		end := i + chunkSize
		if end > len(comments) {
			end = len(comments)
		}
		body := ""
		if i == 0 {
			body = fmt.Sprintf("### 🧪 Unbx Code Quarantine\nDetected %d architecture policy violation(s). Suggested fixes are listed below.", totalViolations)
		}
		reviewPayload := models.GitHubReviewRequest{
			Body:     body,
			Event:    "COMMENT",
			CommitID: commitSHA,
			Comments: comments[i:end],
		}
		client.PrSuggest(ctx, reviewPayload, githubToken, repoSlug, len(comments[i:end]), prNumber)
	}
	// Fail the CI pipeline when violations are found
	os.Exit(1)
}
