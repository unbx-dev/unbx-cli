package models

type BulkScanRequest struct {
	GithubRepositoryID string        `json:"github_repository_id"`
	Files              []FilePayload `json:"files"`
}

type FilePayload struct {
	Path            string `json:"path"`
	EncryptedSource string `json:"encrypted_source"`
	LangName        string `json:"lang_name"`
}

type BulkScanResponse struct {
	Violations []Violation `json:"violations"`
}

type Violation struct {
	RuleTitle    string `json:"rule_title"`
	FilePath     string `json:"file_path"`     // path of the file containing the violation
	StartLine    int    `json:"start_line"`    // violation start line (for GitHub API)
	EndLine      int    `json:"end_line"`      // violation end line (for GitHub API)
	Message      string `json:"message"`       // warning message shown to the developer
	SuggestedFix string `json:"suggested_fix"` // fully-assembled replacement code string built by the backend
}
