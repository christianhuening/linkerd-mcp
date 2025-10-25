package testutil

import (
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
)

// GetTextFromResult extracts text content from an MCP CallToolResult
func GetTextFromResult(result *mcp.CallToolResult, target *string) error {
	if result == nil || len(result.Content) == 0 {
		*target = ""
		return nil
	}

	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		*target = ""
		return nil
	}

	*target = textContent.Text
	return nil
}

// ParseJSONResult parses JSON from MCP result text content
func ParseJSONResult(result *mcp.CallToolResult, v interface{}) error {
	var text string
	if err := GetTextFromResult(result, &text); err != nil {
		return err
	}

	return json.Unmarshal([]byte(text), v)
}
