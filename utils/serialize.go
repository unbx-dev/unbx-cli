package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	sitter "github.com/smacker/go-tree-sitter"
)

// Anonymized node structure sent to the backend
type AnonymizedNode struct {
	Type       string            `json:"type"`        // Tree-sitter node type (e.g. "identifier", "call_expression")
	ContentHex string            `json:"content_hex"` // hashed identifier (raw string is never included)
	StartByte  uint32            `json:"start_byte"`  // byte offset for local substitution
	EndByte    uint32            `json:"end_byte"`    // byte offset for local substitution
	StartRow   uint32            `json:"start_row"`   // 0-based line number in source (for violation line mapping)
	EndRow     uint32            `json:"end_row"`     // 0-based line number in source
	Children   []*AnonymizedNode `json:"children"`    // child nodes
}

// Salt for making proprietary source code irreversible (ideally sourced from an env var)
const anonymizeSalt = "synk-secure-salt-2026"

type CodeSerializer struct {
}

func NewCodeSerializer() *CodeSerializer {
	return &CodeSerializer{}
}

func (c *CodeSerializer) HashIdentifier(text string) string {
	hasher := sha256.New()
	hasher.Write([]byte(text + anonymizeSalt))
	return hex.EncodeToString(hasher.Sum(nil))
}

func (c *CodeSerializer) WalkAndAnonymize(node *sitter.Node, sourceByte []byte) *AnonymizedNode {
	nodeType := node.Type()
	anonNode := &AnonymizedNode{
		Type:      nodeType,
		StartByte: node.StartByte(),
		EndByte:   node.EndByte(),
		StartRow:  node.StartPoint().Row,
		EndRow:    node.EndPoint().Row,
		Children:  make([]*AnonymizedNode, 0, node.ChildCount()),
	}
	switch nodeType {
	case "identifier", "field_identifier", "type_identifier":
		// Hash function names, variable names, and type names to track structural matches
		rawText := string(sourceByte[node.StartByte():node.EndByte()])
		anonNode.ContentHex = c.HashIdentifier(rawText)

	case "string_literal", "string", "number", "integer_literal":
		// String contents and numeric literals carry the highest leak risk — replace completely
		anonNode.ContentHex = c.HashIdentifier("[REDACTED_LITERAL]")

	default:
		// Control structures (if, for, func, etc.) and composite nodes carry no text
		anonNode.ContentHex = ""
	}
	// Recursively process child nodes
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		anonNode.Children = append(anonNode.Children, c.WalkAndAnonymize(child, sourceByte))
	}
	return anonNode
}

func (c *CodeSerializer) SerializeAndAnonymize(root *sitter.Node, sourceByte []byte) string {
	if root == nil {
		return "{}"
	}
	anonymizedRoot := c.WalkAndAnonymize(root, sourceByte)
	jsonBytes, err := json.Marshal(anonymizedRoot)
	if err != nil {
		return "{}"
	}
	return string(jsonBytes)
}
