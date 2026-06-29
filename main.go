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

	// Build a set of valid diff line numbers per file so we only post inline
	// comments on lines that actually appear in the PR diff. GitHub returns 422
	// "Line could not be resolved" for any line outside the diff hunks.
	validDiffLines := make(map[string]map[int]bool, len(changeFiles))
	for _, f := range changeFiles {
		validDiffLines[f.Path] = utils.ParseDiffValidLines(f.PatchCode)
	}

	comments := make([]models.GitHubDraftComment, 0, len(scanResponse.Violations))
	for _, violation := range scanResponse.Violations {
		fileLines, ok := validDiffLines[violation.FilePath]
		if !ok || !fileLines[violation.EndLine] {
			// EndLine is not in the diff — skip to avoid 422
			continue
		}
		commentBody := fmt.Sprintf(
			"### 🚨 Unbx Quarantine Alert: [%s]\n%s\n\n```suggestion\n%s\n```",
			violation.RuleTitle,
			violation.Message,
			violation.SuggestedFix,
		)
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

	reviewPayload := models.GitHubReviewRequest{
		Body:     fmt.Sprintf("### 🧪 Unbx Code Quarantine\nDetected %d architecture policy violation(s). Suggested fixes are listed below.", len(scanResponse.Violations)),
		Event:    "COMMENT",
		CommitID: commitSHA,
		Comments: comments,
	}

	if client.PrSuggest(ctx, reviewPayload, githubToken, repoSlug, len(scanResponse.Violations), prNumber) {
		// Fail the CI pipeline when violations are found
		os.Exit(1)
	}
}
