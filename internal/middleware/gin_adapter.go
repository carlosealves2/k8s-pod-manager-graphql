package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GinHTTPRequest adapts gin.Context to HTTPRequest interface
type GinHTTPRequest struct {
	ctx *gin.Context
}

func (r *GinHTTPRequest) Method() string {
	return r.ctx.Request.Method
}

func (r *GinHTTPRequest) Path() string {
	return r.ctx.Request.URL.Path
}

func (r *GinHTTPRequest) Header(key string) string {
	return r.ctx.GetHeader(key)
}

func (r *GinHTTPRequest) Body() []byte {
	if r.ctx.Request.Body == nil {
		return nil
	}

	// Try to get cached body from context first
	if body, exists := r.ctx.Get("request_body"); exists {
		if bodyBytes, ok := body.([]byte); ok {
			return bodyBytes
		}
	}

	// For Gin, we need to be careful about reading the body
	// as it can only be read once
	return nil
}

func (r *GinHTTPRequest) IP() string {
	return r.ctx.ClientIP()
}

func (r *GinHTTPRequest) UserAgent() string {
	return r.ctx.GetHeader("User-Agent")
}

func (r *GinHTTPRequest) RequestID() string {
	return r.ctx.GetHeader("X-Request-ID")
}

// GinHTTPResponse adapts gin.Context to HTTPResponse interface
type GinHTTPResponse struct {
	ctx *gin.Context
}

func (r *GinHTTPResponse) SetHeader(key, value string) {
	r.ctx.Header(key, value)
}

func (r *GinHTTPResponse) SetStatus(code int) {
	r.ctx.Status(code)
}

func (r *GinHTTPResponse) Write(data []byte) error {
	_, err := r.ctx.Writer.Write(data)
	return err
}

func (r *GinHTTPResponse) JSON(data interface{}) error {
	r.ctx.JSON(http.StatusOK, data)
	return nil
}

// GinHTTPContext adapts gin.Context to HTTPContext interface
type GinHTTPContext struct {
	ctx      *gin.Context
	request  HTTPRequest
	response HTTPResponse
}

func NewGinHTTPContext(ctx *gin.Context) HTTPContext {
	return &GinHTTPContext{
		ctx:      ctx,
		request:  &GinHTTPRequest{ctx: ctx},
		response: &GinHTTPResponse{ctx: ctx},
	}
}

func (c *GinHTTPContext) Request() HTTPRequest {
	return c.request
}

func (c *GinHTTPContext) Response() HTTPResponse {
	return c.response
}

func (c *GinHTTPContext) Set(key string, value interface{}) {
	c.ctx.Set(key, value)
}

func (c *GinHTTPContext) Get(key string) (interface{}, bool) {
	return c.ctx.Get(key)
}

func (c *GinHTTPContext) Next() error {
	c.ctx.Next()
	return nil
}

// ToGinHandler converts a Middleware to a gin.HandlerFunc
func ToGinHandler(middleware Middleware) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		httpCtx := NewGinHTTPContext(ctx)
		if err := middleware.Handle(httpCtx); err != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "Middleware error",
				"details": err.Error(),
			})
		}
	}
}

// GinMiddlewareAdapter wraps a gin.HandlerFunc to implement the Middleware interface
type GinMiddlewareAdapter struct {
	handler gin.HandlerFunc
}

func NewGinMiddlewareAdapter(handler gin.HandlerFunc) Middleware {
	return &GinMiddlewareAdapter{handler: handler}
}

func (g *GinMiddlewareAdapter) Handle(ctx HTTPContext) error {
	if ginCtx, ok := ctx.(*GinHTTPContext); ok {
		g.handler(ginCtx.ctx)
		return nil
	}
	return ErrIncompatibleContext
}

// Common errors
var (
	ErrIncompatibleContext = &MiddlewareError{
		Code:    "INCOMPATIBLE_CONTEXT",
		Message: "Context type is not compatible with this adapter",
	}
)

// MiddlewareError represents an error that occurred in middleware
type MiddlewareError struct {
	Code    string
	Message string
	Cause   error
}

func (e *MiddlewareError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *MiddlewareError) Unwrap() error {
	return e.Cause
}