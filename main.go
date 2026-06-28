package main

import (
	"context"
	"fmt"
	"unbx/client"
	"unbx/models"
	"unbx/utils"
	"log"
	"os"

	sitter "github.com/smacker/go-tree-sitter"
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

	parser := sitter.NewParser()

	scanRequest := models.BulkScanRequest{
		GithubRepositoryID: repositoryID,
	}
	for _, fileDiff := range changeFiles {
		lang, langName := utils.LanguageForFile(fileDiff.Path)
		if lang == nil {
			continue
		}
		parser.SetLanguage(lang)

		sourceBytes := []byte(fileDiff.PatchCode)
		tree, _ := parser.ParseCtx(ctx, nil, sourceBytes)
		rootNode := tree.RootNode()

		// Hash identifiers and extract structure only — never send raw source code
		anonymizedAST := utils.NewCodeSerializer().SerializeAndAnonymize(rootNode, sourceBytes)

		scanRequest.Files = append(scanRequest.Files, models.FilePayload{
			Path:          fileDiff.Path,
			AnonymizedAST: anonymizedAST,
			LangName:      langName,
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

	reviewPayload := models.GitHubReviewRequest{
		Body:     fmt.Sprintf("### 🧪 Unbx Code Quarantine\nDetected %d architecture policy violation(s). Suggested fixes are listed below.", len(scanResponse.Violations)),
		Event:    "COMMENT", // Leave as a comment review without approving
		CommitID: commitSHA,
		Comments: make([]models.GitHubDraftComment, 0, len(scanResponse.Violations)),
	}
	for _, violation := range scanResponse.Violations {
		commentBody := fmt.Sprintf(
			"### 🚨 Unbx Quarantine Alert: [%s]\n%s\n\n```suggestion\n%s\n```",
			violation.RuleTitle,
			violation.Message,
			violation.SuggestedFix,
		)
		reviewPayload.Comments = append(reviewPayload.Comments, models.GitHubDraftComment{
			Path:      violation.FilePath,
			Body:      commentBody,
			StartLine: violation.StartLine,
			Line:      violation.EndLine,
			Side:      "RIGHT",
			StartSide: "RIGHT",
		})
	}

	if client.PrSuggest(ctx, reviewPayload, githubToken, repoSlug, len(scanResponse.Violations), prNumber) {
		// Fail the CI pipeline when violations are found
		os.Exit(1)
	}
}
