package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mark3labs/mcp-go/mcp"
)

func (tm *ToolsManager) HandleToolWhoami(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

	validatedJwt := request.Header.Get(tm.dependencies.AppCtx.Config.Middleware.JWT.Validation.ForwardedHeader)

	// Debug information
	debugInfo := fmt.Sprintf(`🔍 **Debug Information**

**JWT Header Name:** %s
**JWT Present:** %t
**JWT Length:** %d
**Full JWT:** %s

---

`, tm.dependencies.AppCtx.Config.Middleware.JWT.Validation.ForwardedHeader,
		validatedJwt != "",
		len(validatedJwt),
		validatedJwt)

	if validatedJwt == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: debugInfo + "❌ **Error:** JWT is empty. User information is not available\n\n**Possible causes:**\n- JWT validation is disabled in config\n- JWT header not being forwarded\n- Server not receiving JWT from client",
				},
			},
		}, nil
	}

	// Try to parse JWT
	token, _, err := new(jwt.Parser).ParseUnverified(validatedJwt, jwt.MapClaims{})
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: debugInfo + fmt.Sprintf("❌ **Error parsing JWT:** %v\n\nThe JWT might be malformed or use an unsupported format.", err),
				},
			},
		}, nil
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: debugInfo + "❌ **Error:** Unable to parse JWT claims\n\nThe JWT structure might be invalid.",
				},
			},
		}, nil
	}

	// Build user information
	var userInfo strings.Builder
	userInfo.WriteString(debugInfo)
	userInfo.WriteString("✅ **JWT Successfully Decoded!**\n\n")
	userInfo.WriteString("👤 **User Information**\n\n")

	// Standard JWT claims
	if sub, ok := claims["sub"].(string); ok && sub != "" {
		userInfo.WriteString(fmt.Sprintf("**Subject (User ID):** %s\n", sub))
	}

	if name, ok := claims["name"].(string); ok && name != "" {
		userInfo.WriteString(fmt.Sprintf("**Name:** %s\n", name))
	}

	if email, ok := claims["email"].(string); ok && email != "" {
		userInfo.WriteString(fmt.Sprintf("**Email:** %s\n", email))
	}

	if username, ok := claims["preferred_username"].(string); ok && username != "" {
		userInfo.WriteString(fmt.Sprintf("**Username:** %s\n", username))
	}

	// Issuer information
	if iss, ok := claims["iss"].(string); ok && iss != "" {
		userInfo.WriteString(fmt.Sprintf("**Issuer:** %s\n", iss))
	}

	// Audience
	if aud := claims["aud"]; aud != nil {
		switch a := aud.(type) {
		case string:
			userInfo.WriteString(fmt.Sprintf("**Audience:** %s\n", a))
		case []interface{}:
			audiences := make([]string, len(a))
			for i, v := range a {
				if s, ok := v.(string); ok {
					audiences[i] = s
				}
			}
			userInfo.WriteString(fmt.Sprintf("**Audience:** %s\n", strings.Join(audiences, ", ")))
		}
	}

	// Token timing information
	userInfo.WriteString("\n⏰ **Token Information**\n\n")

	if iat, ok := claims["iat"].(float64); ok {
		issuedAt := time.Unix(int64(iat), 0)
		userInfo.WriteString(fmt.Sprintf("**Issued At:** %s\n", issuedAt.Format("2006-01-02 15:04:05 MST")))
	}

	if exp, ok := claims["exp"].(float64); ok {
		expiresAt := time.Unix(int64(exp), 0)
		userInfo.WriteString(fmt.Sprintf("**Expires At:** %s\n", expiresAt.Format("2006-01-02 15:04:05 MST")))

		if time.Now().Before(expiresAt) {
			timeUntilExpiry := time.Until(expiresAt)
			userInfo.WriteString(fmt.Sprintf("**Time Until Expiry:** %s\n", timeUntilExpiry.Round(time.Second)))
		} else {
			userInfo.WriteString("⚠️ **Token is expired**\n")
		}
	}

	// Custom claims section
	customClaims := make(map[string]interface{})
	standardClaims := map[string]bool{
		"sub": true, "name": true, "email": true, "preferred_username": true,
		"iss": true, "aud": true, "iat": true, "exp": true, "nbf": true, "jti": true,
	}

	for key, value := range claims {
		if !standardClaims[key] {
			customClaims[key] = value
		}
	}

	if len(customClaims) > 0 {
		userInfo.WriteString("\n🔧 **Custom Claims**\n\n")
		for key, value := range customClaims {
			// Pretty print JSON values
			if jsonValue, err := json.MarshalIndent(value, "", "  "); err == nil {
				userInfo.WriteString(fmt.Sprintf("**%s:** %s\n", key, string(jsonValue)))
			} else {
				userInfo.WriteString(fmt.Sprintf("**%s:** %v\n", key, value))
			}
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: userInfo.String(),
			},
		},
	}, nil
}
