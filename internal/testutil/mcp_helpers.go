package testutil

import (
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
)

// GetTextFromResult extracts text content from an MCP CallToolResult
func GetTextFromResult(result *mcp.CallToolResult) (string, error) {
	if result == nil || len(result.Content) == 0 {
		return "", nil
	}

	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		return "", nil
	}

	return textContent.Text, nil
}

// ParseJSONResult parses JSON from MCP result text content
func ParseJSONResult(result *mcp.CallToolResult, v interface{}) error {
	text, err := GetTextFromResult(result)
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(text), v)
}
