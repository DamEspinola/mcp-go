package handlers

import "mcp-go/internal/globals"

type HandlersManagerDependencies struct {
	AppCtx *globals.ApplicationContext
}

type HandlersManager struct {
	dependencies HandlersManagerDependencies
}

func NewHandlersManager(deps HandlersManagerDependencies) *HandlersManager {
	return &HandlersManager{
		dependencies: deps,
	}
}
