package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mark3labs/mcp-go/mcp"
)

func (tm *ToolsManager) HandleToolGenerateJWT(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Default values
	name := "Test User"
	email := "test@example.com"
	username := "testuser"

	// Override with provided parameters if available
	if request.Params.Arguments != nil {
		if argsMap, ok := request.Params.Arguments.(map[string]interface{}); ok {
			if nameVal, ok := argsMap["name"].(string); ok && nameVal != "" {
				name = nameVal
			}
			if emailVal, ok := argsMap["email"].(string); ok && emailVal != "" {
				email = emailVal
			}
			if usernameVal, ok := argsMap["username"].(string); ok && usernameVal != "" {
				username = usernameVal
			}
		}
	}

	// Create JWT claims
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":                "mcp-go-dev",
		"sub":                "test-user-12345",
		"aud":                "mcp-go",
		"iat":                now.Unix(),
		"exp":                now.Add(24 * time.Hour).Unix(), // Expires in 24 hours
		"name":               name,
		"email":              email,
		"preferred_username": username,
		"groups":             []string{"users", "developers"},
		"custom_claim":       "This is a test JWT for development",
	}

	// Create token with HS256 (for development only)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token with a simple secret (for development only)
	secretKey := []byte("development-secret-key-change-in-production")
	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error generating JWT: %v", err),
				},
			},
		}, nil
	}

	// Build response
	response := fmt.Sprintf(`🔑 **JWT Generated Successfully**

**Token:**
%s

**Claims:**
- **Name:** %s
- **Email:** %s  
- **Username:** %s
- **Subject:** test-user-12345
- **Issuer:** mcp-go-dev
- **Audience:** mcp-go
- **Expires:** %s (24 hours from now)
- **Groups:** users, developers

**Usage:**
Set this as the JWT environment variable in your Claude Desktop config:

`+"```json5"+`
{
  "mcpServers": {
    "local-proxy-remote": {
      "command": "npx",
      "args": [
        "mcp-remote",
        "http://localhost:8080/mcp",
        "--transport",
        "http-only",
        "--header",
        "Authorization: Bearer ${JWT}",
        "--header", 
        "X-Validated-Jwt: ${JWT}"
      ],
      "env": {
        "JWT": "%s"
      }
    }
  }
}
`+"```"+`

⚠️ **Note:** This is a development JWT only. In production, use a proper OAuth provider like Keycloak.`,
		tokenString,
		name,
		email,
		username,
		now.Add(24*time.Hour).Format("2006-01-02 15:04:05 MST"),
		tokenString)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: response,
			},
		},
	}, nil
}
