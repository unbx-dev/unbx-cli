package models

// Model for sending AST structure to the API
type AstStructurePayload struct {
	RepositoryID string
	LangName     string
	SourceBytes  string
}
