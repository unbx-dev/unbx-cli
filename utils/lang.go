package utils

import (
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/bash"
	"github.com/smacker/go-tree-sitter/c"
	"github.com/smacker/go-tree-sitter/cpp"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/php"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/ruby"
	"github.com/smacker/go-tree-sitter/rust"
	"github.com/smacker/go-tree-sitter/typescript/tsx"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

func LanguageForFile(path string) (*sitter.Language, string) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return golang.GetLanguage(), "go"
	case ".js", ".jsx", ".mjs", ".cjs":
		return javascript.GetLanguage(), "javascript"
	case ".ts":
		return typescript.GetLanguage(), "typescript"
	case ".tsx":
		return tsx.GetLanguage(), "tsx"
	case ".py":
		return python.GetLanguage(), "python"
	case ".rb":
		return ruby.GetLanguage(), "ruby"
	case ".rs":
		return rust.GetLanguage(), "rust"
	case ".java":
		return java.GetLanguage(), "java"
	case ".php":
		return php.GetLanguage(), "php"
	case ".c", ".h":
		return c.GetLanguage(), "c"
	case ".cpp", ".cc", ".cxx", ".hpp":
		return cpp.GetLanguage(), "cpp"
	case ".sh", ".bash":
		return bash.GetLanguage(), "bash"
	default:
		return nil, ""
	}
}
