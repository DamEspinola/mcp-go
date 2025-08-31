package tools

import (
	"context"
	"fmt"
	"os"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/joho/godotenv"
	"github.com/mark3labs/mcp-go/mcp"
)

// TODO: Implementar clean code aqui

// DatabaseConnection holds connection information for a database
type DatabaseConnection struct {
	Driver     string
	Connection *gorm.DB
}

// createGormConnection creates a new GORM database connection
func createGormConnection(driver, connectionString string) (*gorm.DB, error) {
	var db *gorm.DB
	var err error

	switch driver {
	case "postgres":
		db, err = gorm.Open(postgres.Open(connectionString), &gorm.Config{})
	default:
		return nil, fmt.Errorf("unsupported driver: %s. Currently only postgres is supported", driver)
	}

	if err != nil {
		return nil, err
	}

	// Test the connection
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	if err := sqlDB.Ping(); err != nil {
		return nil, err
	}

	return db, nil
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

	// Validar que la consulta sea SELECT o INSERT por seguridad
	normalizedQuery := strings.TrimSpace(strings.ToUpper(query))
	if !(strings.HasPrefix(normalizedQuery, "SELECT") || strings.HasPrefix(normalizedQuery, "INSERT")) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "❌ **Error:** Solo se permiten consultas SELECT o INSERT por razones de seguridad",
				},
			},
			IsError: true,
		}, nil
	}

	// Obtener la conexión a la base de datos
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

	// Manejar SELECT o INSERT de manera diferente
	if strings.HasPrefix(normalizedQuery, "SELECT") {
		return tm.handleSelectQuery(ctx, connectionName, dbConn, query)
	} else if strings.HasPrefix(normalizedQuery, "INSERT") {
		return tm.handleInsertQuery(ctx, connectionName, dbConn, query)
	}

	// Este return nunca debería ejecutarse debido a la validación anterior
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: "❌ **Error:** Tipo de consulta no soportado",
			},
		},
		IsError: true,
	}, nil
}

func (tm *ToolsManager) handleSelectQuery(ctx context.Context, connectionName string, dbConn *DatabaseConnection, query string) (*mcp.CallToolResult, error) {
	// Get the underlying sql.DB from GORM
	sqlDB, err := dbConn.Connection.DB()
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("❌ **Error getting SQL DB:** %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// Ejecutar la consulta SELECT usando GORM's raw SQL capability
	rows, err := sqlDB.QueryContext(ctx, query)
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

	// Obtener información de columnas
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

	// Preparar resultado
	var result strings.Builder
	result.WriteString(fmt.Sprintf("✅ **Database Query Results** (Connection: %s, Driver: %s)\n\n", connectionName, dbConn.Driver))
	result.WriteString(fmt.Sprintf("**Query:** `%s`\n\n", query))

	// Crear encabezado de tabla
	result.WriteString("| ")
	for _, col := range columns {
		result.WriteString(fmt.Sprintf("%s | ", col))
	}
	result.WriteString("\n")

	// Crear separador de tabla
	result.WriteString("|")
	for range columns {
		result.WriteString("---|")
	}
	result.WriteString("\n")

	// Procesar filas
	rowCount := 0
	maxRows := 100 // Limitar resultados por rendimiento

	for rows.Next() && rowCount < maxRows {
		// Crear slice para contener los valores
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// Escanear la fila en los punteros de valor
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

		// Convertir valores a strings y construir fila
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
			// Escapar caracteres pipe y limitar longitud
			str = strings.ReplaceAll(str, "|", "\\|")
			if len(str) > 50 {
				str = str[:47] + "..."
			}
			result.WriteString(fmt.Sprintf("%s | ", str))
		}
		result.WriteString("\n")
		rowCount++
	}

	// Verificar errores de iteración
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

	// Agregar resumen
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

func (tm *ToolsManager) handleInsertQuery(ctx context.Context, connectionName string, dbConn *DatabaseConnection, query string) (*mcp.CallToolResult, error) {
	// Get the underlying sql.DB from GORM
	sqlDB, err := dbConn.Connection.DB()
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("❌ **Error getting SQL DB:** %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// Ejecutar la consulta INSERT
	result, err := sqlDB.ExecContext(ctx, query)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("❌ **Database Insert Error:**\n\n%v\n\n**Query:** %s", err, query),
				},
			},
			IsError: true,
		}, nil
	}

	// ...existing code for handling result...
	// Obtener número de filas afectadas
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		rowsAffected = -1 // Indicar que no se pudo obtener el número
	}

	// Intentar obtener el último ID insertado (útil para tablas con auto-increment)
	var lastInsertInfo string
	if lastInsertID, err := result.LastInsertId(); err == nil && lastInsertID > 0 {
		lastInsertInfo = fmt.Sprintf("\n- **Último ID insertado:** %d", lastInsertID)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("✅ **INSERT ejecutado correctamente** (Connection: %s, Driver: %s)\n\n**Query:** `%s`\n\n📊 **Resultado:**\n- **Filas afectadas:** %d%s",
					connectionName, dbConn.Driver, query, rowsAffected, lastInsertInfo),
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
	}

	if !validDrivers[driver] {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "❌ **Error:** Invalid driver. Currently only 'postgres' is supported with GORM",
				},
			},
			IsError: true,
		}, nil
	}

	// Close existing connection if it exists
	if existingConn, exists := dbConnections[connectionName]; exists {
		if sqlDB, err := existingConn.Connection.DB(); err == nil {
			sqlDB.Close()
		}
	}

	// Create new GORM connection
	db, err := createGormConnection(driver, connectionString)
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

	// Store the connection (the connection test is already done in createGormConnection)
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
		// Test connection using GORM
		if sqlDB, err := conn.Connection.DB(); err != nil {
			status = "❌ Disconnected"
		} else if err := sqlDB.Ping(); err != nil {
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
		if sqlDB, err := existingConn.Connection.DB(); err == nil {
			sqlDB.Close()
		}
	}

	// Create new GORM connection
	db, err := createGormConnection("postgres", databaseURL)
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

	// Store the connection (the connection test is already done in createGormConnection)
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
