package models

type GitHubReviewRequest struct {
	Body     string               `json:"body"`      // overall summary message for the review
	Event    string               `json:"event"`     // "COMMENT" (or "REQUEST_CHANGES")
	CommitID string               `json:"commit_id"` // target commit SHA
	Comments []GitHubDraftComment `json:"comments"`  // all suggestions packed into this slice
}

type GitHubDraftComment struct {
	Path      string `json:"path"`
	Body      string `json:"body"` // Markdown containing ```suggestion
	StartLine int    `json:"start_line,omitempty"`
	Line      int    `json:"line"`       // end line
	Side      string `json:"side"`       // always "RIGHT"
	StartSide string `json:"start_side,omitempty"`
}

type PRFileDiff struct {
	Path      string
	PatchCode string
}
