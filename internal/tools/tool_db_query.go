package tools

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql" // MySQL driver
	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/mark3labs/mcp-go/mcp"
	_ "modernc.org/sqlite" // SQLite driver
)

// DatabaseConnection holds connection information for a database
type DatabaseConnection struct {
	Driver     string
	Connection *sql.DB
}

// dbConnections stores active database connections
var dbConnections = make(map[string]*DatabaseConnection)

func (tm *ToolsManager) HandleToolDatabaseQuery(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := request.GetArguments()

	// Get required parameters
	connectionName, ok := arguments["connection_name"].(string)
	if !ok || connectionName == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "❌ **Error:** connection_name parameter is required and must be a string",
				},
			},
			IsError: true,
		}, nil
	}

	query, ok := arguments["query"].(string)
	if !ok || query == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "❌ **Error:** query parameter is required and must be a string",
				},
			},
			IsError: true,
		}, nil
	}

	// Validate that the query is a SELECT statement for security
	normalizedQuery := strings.TrimSpace(strings.ToUpper(query))
	if !strings.HasPrefix(normalizedQuery, "SELECT") {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "❌ **Error:** Only SELECT queries are allowed for security reasons",
				},
			},
			IsError: true,
		}, nil
	}

	// Get the database connection
	dbConn, exists := dbConnections[connectionName]
	if !exists {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("❌ **Error:** Database connection '%s' not found. Please create the connection first using the connect_database tool.", connectionName),
				},
			},
			IsError: true,
		}, nil
	}

	// Execute the query
	rows, err := dbConn.Connection.QueryContext(ctx, query)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("❌ **Database Query Error:**\n\n%v\n\n**Query:** %s", err, query),
				},
			},
			IsError: true,
		}, nil
	}
	defer rows.Close()

	// Get column information
	columns, err := rows.Columns()
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("❌ **Error getting column information:** %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// Prepare result
	var result strings.Builder
	result.WriteString(fmt.Sprintf("✅ **Database Query Results** (Connection: %s, Driver: %s)\n\n", connectionName, dbConn.Driver))
	result.WriteString(fmt.Sprintf("**Query:** `%s`\n\n", query))

	// Create table header
	result.WriteString("| ")
	for _, col := range columns {
		result.WriteString(fmt.Sprintf("%s | ", col))
	}
	result.WriteString("\n")

	// Create table separator
	result.WriteString("|")
	for range columns {
		result.WriteString("---|")
	}
	result.WriteString("\n")

	// Process rows
	rowCount := 0
	maxRows := 100 // Limit results for performance

	for rows.Next() && rowCount < maxRows {
		// Create a slice to hold the values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// Scan the row into the value pointers
		if err := rows.Scan(valuePtrs...); err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("❌ **Error scanning row %d:** %v", rowCount+1, err),
					},
				},
				IsError: true,
			}, nil
		}

		// Convert values to strings and build row
		result.WriteString("| ")
		for _, val := range values {
			var str string
			if val == nil {
				str = "NULL"
			} else {
				switch v := val.(type) {
				case []byte:
					str = string(v)
				default:
					str = fmt.Sprintf("%v", v)
				}
			}
			// Escape pipe characters and limit length
			str = strings.ReplaceAll(str, "|", "\\|")
			if len(str) > 50 {
				str = str[:47] + "..."
			}
			result.WriteString(fmt.Sprintf("%s | ", str))
		}
		result.WriteString("\n")
		rowCount++
	}

	// Check for iteration errors
	if err := rows.Err(); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("❌ **Error during row iteration:** %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// Add summary
	result.WriteString(fmt.Sprintf("\n📊 **Summary:**\n- **Rows returned:** %d", rowCount))
	if rowCount >= maxRows {
		result.WriteString(fmt.Sprintf(" (limited to %d rows)", maxRows))
	}
	result.WriteString(fmt.Sprintf("\n- **Columns:** %d (%s)", len(columns), strings.Join(columns, ", ")))

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: result.String(),
			},
		},
	}, nil
}

func (tm *ToolsManager) HandleToolDatabaseConnect(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := request.GetArguments()

	// Get required parameters
	connectionName, ok := arguments["connection_name"].(string)
	if !ok || connectionName == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "❌ **Error:** connection_name parameter is required and must be a string",
				},
			},
			IsError: true,
		}, nil
	}

	driver, ok := arguments["driver"].(string)
	if !ok || driver == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "❌ **Error:** driver parameter is required and must be a string (postgres, mysql, sqlite)",
				},
			},
			IsError: true,
		}, nil
	}

	connectionString, ok := arguments["connection_string"].(string)
	if !ok || connectionString == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "❌ **Error:** connection_string parameter is required and must be a string",
				},
			},
			IsError: true,
		}, nil
	}

	// Validate driver
	validDrivers := map[string]bool{
		"postgres": true,
		"mysql":    true,
		"sqlite":   true,
	}

	if !validDrivers[driver] {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "❌ **Error:** Invalid driver. Supported drivers: postgres, mysql, sqlite",
				},
			},
			IsError: true,
		}, nil
	}

	// Close existing connection if it exists
	if existingConn, exists := dbConnections[connectionName]; exists {
		existingConn.Connection.Close()
	}

	// Create new connection
	db, err := sql.Open(driver, connectionString)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("❌ **Error connecting to database:**\n\n%v\n\n**Driver:** %s\n**Connection:** %s", err, driver, connectionString),
				},
			},
			IsError: true,
		}, nil
	}

	// Test the connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("❌ **Error pinging database:**\n\n%v\n\n**Driver:** %s\n**Connection:** %s", err, driver, connectionString),
				},
			},
			IsError: true,
		}, nil
	}

	// Store the connection
	dbConnections[connectionName] = &DatabaseConnection{
		Driver:     driver,
		Connection: db,
	}

	// Prepare success message with connection examples
	var examples strings.Builder
	examples.WriteString("\n\n📝 **Example Queries:**\n\n")

	switch driver {
	case "postgres":
		examples.WriteString("```sql\n")
		examples.WriteString("-- List all tables\n")
		examples.WriteString("SELECT tablename FROM pg_tables WHERE schemaname = 'public';\n\n")
		examples.WriteString("-- Get table structure\n")
		examples.WriteString("SELECT column_name, data_type, is_nullable FROM information_schema.columns WHERE table_name = 'your_table';\n\n")
		examples.WriteString("-- Sample data query\n")
		examples.WriteString("SELECT * FROM your_table LIMIT 10;\n")
		examples.WriteString("```")
	case "mysql":
		examples.WriteString("```sql\n")
		examples.WriteString("-- List all tables\n")
		examples.WriteString("SHOW TABLES;\n\n")
		examples.WriteString("-- Get table structure\n")
		examples.WriteString("DESCRIBE your_table;\n\n")
		examples.WriteString("-- Sample data query\n")
		examples.WriteString("SELECT * FROM your_table LIMIT 10;\n")
		examples.WriteString("```")
	case "sqlite":
		examples.WriteString("```sql\n")
		examples.WriteString("-- List all tables\n")
		examples.WriteString("SELECT name FROM sqlite_master WHERE type='table';\n\n")
		examples.WriteString("-- Get table structure\n")
		examples.WriteString("PRAGMA table_info(your_table);\n\n")
		examples.WriteString("-- Sample data query\n")
		examples.WriteString("SELECT * FROM your_table LIMIT 10;\n")
		examples.WriteString("```")
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("✅ **Database Connected Successfully!**\n\n**Connection Name:** %s\n**Driver:** %s\n**Status:** Active\n\nYou can now use the `database_query` tool to execute SELECT queries on this connection.%s", connectionName, driver, examples.String()),
			},
		},
	}, nil
}

func (tm *ToolsManager) HandleToolDatabaseList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if len(dbConnections) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "📋 **No database connections found**\n\nUse the `connect_database` tool to create a connection first.",
				},
			},
		}, nil
	}

	var result strings.Builder
	result.WriteString("📋 **Active Database Connections**\n\n")
	result.WriteString("| Connection Name | Driver | Status |\n")
	result.WriteString("|---|---|---|\n")

	for name, conn := range dbConnections {
		status := "Active"
		if err := conn.Connection.Ping(); err != nil {
			status = "❌ Disconnected"
		} else {
			status = "✅ Active"
		}
		result.WriteString(fmt.Sprintf("| %s | %s | %s |\n", name, conn.Driver, status))
	}

	result.WriteString("\n💡 **Tip:** Use `database_query` with any of these connection names to execute queries.")

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: result.String(),
			},
		},
	}, nil
}

func (tm *ToolsManager) HandleToolDatabaseConnectFromEnv(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := request.GetArguments()

	// Get connection name parameter
	connectionName, ok := arguments["connection_name"].(string)
	if !ok || connectionName == "" {
		connectionName = "default" // Default connection name
	}

	// Load .env file
	if err := godotenv.Load(); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "⚠️ **Warning:** Could not load .env file. Make sure you have a .env file in your project root with DATABASE_URL defined.\n\nExample .env content:\n```\nDATABASE_URL=postgres://user:password@localhost:5432/dbname?sslmode=disable\n```",
				},
			},
			IsError: true,
		}, nil
	}

	// Get DATABASE_URL from environment
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "❌ **Error:** DATABASE_URL environment variable not found.\n\nPlease add DATABASE_URL to your .env file:\n```\nDATABASE_URL=postgres://user:password@localhost:5432/dbname?sslmode=disable\n```",
				},
			},
			IsError: true,
		}, nil
	}

	// Close existing connection if it exists
	if existingConn, exists := dbConnections[connectionName]; exists {
		existingConn.Connection.Close()
	}

	// Create new connection
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("❌ **Error connecting to database:**\n\n%v\n\n**Database URL:** %s", err, databaseURL),
				},
			},
			IsError: true,
		}, nil
	}

	// Test the connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("❌ **Error pinging database:**\n\n%v\n\n**Database URL:** %s", err, databaseURL),
				},
			},
			IsError: true,
		}, nil
	}

	// Store the connection
	dbConnections[connectionName] = &DatabaseConnection{
		Driver:     "postgres",
		Connection: db,
	}

	// Prepare success message
	maskedURL := maskDatabaseURL(databaseURL)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("✅ **Database Connected Successfully from Environment!**\n\n**Connection Name:** %s\n**Driver:** postgres\n**Database URL:** %s\n**Status:** Active\n\nYou can now use the `database_query` tool to execute SELECT queries on this connection.\n\n📝 **Example Queries:**\n\n```sql\n-- List all tables\nSELECT tablename FROM pg_tables WHERE schemaname = 'public';\n\n-- Get table structure\nSELECT column_name, data_type, is_nullable FROM information_schema.columns WHERE table_name = 'your_table';\n\n-- Sample data query\nSELECT * FROM your_table LIMIT 10;\n```", connectionName, maskedURL),
			},
		},
	}, nil
}

// maskDatabaseURL oculta la contraseña en la URL para mostrarla de forma segura
func maskDatabaseURL(url string) string {
	// Buscar el patrón usuario:contraseña@
	parts := strings.Split(url, "@")
	if len(parts) < 2 {
		return url // No hay @ en la URL, retornar tal como está
	}

	// Dividir la primera parte para obtener usuario:contraseña
	beforeAt := parts[0]
	userPassParts := strings.Split(beforeAt, "://")
	if len(userPassParts) < 2 {
		return url // No hay :// en la URL
	}

	protocol := userPassParts[0] + "://"
	userPass := userPassParts[1]

	// Dividir usuario:contraseña
	credParts := strings.Split(userPass, ":")
	if len(credParts) < 2 {
		return url // No hay : en las credenciales
	}

	username := credParts[0]

	// Construir URL enmascarada
	maskedURL := fmt.Sprintf("%s%s:***@%s", protocol, username, strings.Join(parts[1:], "@"))
	return maskedURL
}
