package tools

import (
	"mcp-go/internal/globals"
	"mcp-go/internal/middlewares"

	//
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type ToolsManagerDependencies struct {
	AppCtx *globals.ApplicationContext

	McpServer   *server.MCPServer
	Middlewares []middlewares.ToolMiddleware
}

type ToolsManager struct {
	dependencies ToolsManagerDependencies
}

func NewToolsManager(deps ToolsManagerDependencies) *ToolsManager {
	return &ToolsManager{
		dependencies: deps,
	}
}

func (tm *ToolsManager) AddTools() {

	// 1. Describe a tool, then add it
	tool := mcp.NewTool("hello_world",
		mcp.WithDescription("Say hello to someone"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the person to greet"),
		),
	)
	tm.dependencies.McpServer.AddTool(tool, tm.HandleToolHello)

	// 2. Describe and add another tool
	tool = mcp.NewTool("whoami",
		mcp.WithDescription("Get detailed information about the current user from their JWT token"),
	)
	tm.dependencies.McpServer.AddTool(tool, tm.HandleToolWhoami)

	// 3. JWT Generator tool for development
	tool = mcp.NewTool("generate_jwt",
		mcp.WithDescription("Generate a development JWT token for testing MCP server authentication"),
		mcp.WithString("name",
			mcp.Description("Name for the JWT claims (optional, default: 'Test User')"),
		),
		mcp.WithString("email",
			mcp.Description("Email for the JWT claims (optional, default: 'test@example.com')"),
		),
		mcp.WithString("username",
			mcp.Description("Username for the JWT claims (optional, default: 'testuser')"),
		),
	)
	tm.dependencies.McpServer.AddTool(tool, tm.HandleToolGenerateJWT)

	// 4. Database connection tool
	tool = mcp.NewTool("connect_database",
		mcp.WithDescription("Connect to a PostgreSQL database for querying"),
		mcp.WithString("connection_name",
			mcp.Required(),
			mcp.Description("A unique name to identify this database connection"),
		),
		mcp.WithString("driver",
			mcp.Required(),
			mcp.Description("Database driver (use 'postgres' for PostgreSQL)"),
		),
		mcp.WithString("connection_string",
			mcp.Required(),
			mcp.Description("PostgreSQL connection string (e.g., 'postgres://user:password@localhost/dbname?sslmode=disable')"),
		),
	)
	tm.dependencies.McpServer.AddTool(tool, tm.HandleToolDatabaseConnect)

	// 5. Database query tool
	tool = mcp.NewTool("database_query",
		mcp.WithDescription("Execute SELECT queries on a connected PostgreSQL database"),
		mcp.WithString("connection_name",
			mcp.Required(),
			mcp.Description("Name of the database connection to use"),
		),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("SELECT query to execute (only SELECT statements are allowed for security)"),
		),
	)
	tm.dependencies.McpServer.AddTool(tool, tm.HandleToolDatabaseQuery)

	// 6. List database connections tool
	tool = mcp.NewTool("list_database_connections",
		mcp.WithDescription("List all active database connections and their status"),
	)
	tm.dependencies.McpServer.AddTool(tool, tm.HandleToolDatabaseList)

	// 7. Connect database from environment tool
	tool = mcp.NewTool("connect_database_env",
		mcp.WithDescription("Connect to PostgreSQL database using DATABASE_URL from .env file"),
		mcp.WithString("connection_name",
			mcp.Description("Name to identify this database connection (optional, default: 'default')"),
		),
	)
	tm.dependencies.McpServer.AddTool(tool, tm.HandleToolDatabaseConnectFromEnv)
}
