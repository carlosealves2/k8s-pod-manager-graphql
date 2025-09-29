package handlers

import (
	"context"

	"github.com/gin-gonic/gin"
)

// GinContextAdapter adapts gin.Context to our Context interface
type GinContextAdapter struct {
	ctx *gin.Context
}

// NewGinContextAdapter creates a new Gin context adapter
func NewGinContextAdapter(ctx *gin.Context) Context {
	return &GinContextAdapter{ctx: ctx}
}

// JSON sends a JSON response
func (g *GinContextAdapter) JSON(statusCode int, data interface{}) {
	g.ctx.JSON(statusCode, data)
}

// String sends a string response
func (g *GinContextAdapter) String(statusCode int, data string) {
	g.ctx.String(statusCode, data)
}

// Header sets a response header
func (g *GinContextAdapter) Header(key, value string) {
	g.ctx.Header(key, value)
}

// Param gets a URL parameter
func (g *GinContextAdapter) Param(key string) string {
	return g.ctx.Param(key)
}

// Query gets a query parameter
func (g *GinContextAdapter) Query(key string) string {
	return g.ctx.Query(key)
}

// Get gets a header value
func (g *GinContextAdapter) Get(key string) string {
	return g.ctx.GetHeader(key)
}

// BodyParser parses the request body
func (g *GinContextAdapter) BodyParser(out interface{}) error {
	return g.ctx.ShouldBindJSON(out)
}

// Context returns the underlying context
func (g *GinContextAdapter) Context() context.Context {
	return g.ctx.Request.Context()
}

// GinRouterAdapter adapts gin router to our Router interface
type GinRouterAdapter struct {
	group *gin.RouterGroup
}

// NewGinRouterAdapter creates a new Gin router adapter
func NewGinRouterAdapter(group *gin.RouterGroup) Router {
	return &GinRouterAdapter{group: group}
}

// GET registers a GET route
func (g *GinRouterAdapter) GET(path string, handler func(Context)) {
	g.group.GET(path, g.adaptHandler(handler))
}

// POST registers a POST route
func (g *GinRouterAdapter) POST(path string, handler func(Context)) {
	g.group.POST(path, g.adaptHandler(handler))
}

// PUT registers a PUT route
func (g *GinRouterAdapter) PUT(path string, handler func(Context)) {
	g.group.PUT(path, g.adaptHandler(handler))
}

// DELETE registers a DELETE route
func (g *GinRouterAdapter) DELETE(path string, handler func(Context)) {
	g.group.DELETE(path, g.adaptHandler(handler))
}

// adaptHandler converts our handler function to gin handler
func (g *GinRouterAdapter) adaptHandler(handler func(Context)) gin.HandlerFunc {
	return func(c *gin.Context) {
		adapter := NewGinContextAdapter(c)
		handler(adapter)
	}
}