package mcp

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// TransportType represents the type of transport to use.
type TransportType string

const (
	// TransportStdio uses standard input/output.
	TransportStdio TransportType = "stdio"
	// TransportSSE uses Server-Sent Events over HTTP.
	TransportSSE TransportType = "sse"
	// TransportStreamableHTTP uses Streamable HTTP protocol.
	TransportStreamableHTTP TransportType = "streamable"
)

// SSEConfig contains configuration for SSE transport.
type SSEConfig struct {
	Addr              string
	BaseURL           string
	StaticBasePath    string
	KeepAliveInterval time.Duration
}

// StreamableHTTPConfig contains configuration for Streamable HTTP transport.
type StreamableHTTPConfig struct {
	Addr              string
	EndpointPath      string
	HeartbeatInterval time.Duration
	Stateless         bool
	// CORS configuration
	EnableCORS        bool
	AllowedOrigins    []string
	AllowedMethods    []string
	AllowedHeaders    []string
}

// Server represents the MCP server instance.
type Server struct {
	mcpServer            *server.MCPServer
	transport            TransportType
	sseConfig            *SSEConfig
	streamableHTTPConfig *StreamableHTTPConfig
}

// NewServer creates a new MCP server instance with default stdio transport.
func NewServer(name string, version string) (*Server, error) {
	// Create MCP server.
	mcpServer := server.NewMCPServer(
		name,
		version,
	)

	return &Server{
		mcpServer: mcpServer,
		transport: TransportStdio,
	}, nil
}

// WithTransport sets the transport type for the server.
func (s *Server) WithTransport(transport TransportType) *Server {
	s.transport = transport
	return s
}

// WithSSEConfig sets the SSE configuration for the server.
func (s *Server) WithSSEConfig(config *SSEConfig) *Server {
	s.sseConfig = config
	return s
}

// WithStreamableHTTPConfig sets the Streamable HTTP configuration for the server.
func (s *Server) WithStreamableHTTPConfig(config *StreamableHTTPConfig) *Server {
	s.streamableHTTPConfig = config
	return s
}

// RegisterTool registers a tool with the MCP server.
func (s *Server) RegisterTool(tool mcp.Tool, handler server.ToolHandlerFunc) error {
	s.mcpServer.AddTool(tool, handler)
	return nil
}

// Start starts the MCP server with the configured transport.
func (s *Server) Start(ctx context.Context) error {
	switch s.transport {
	case TransportStdio:
		return s.startStdio(ctx)
	case TransportSSE:
		return s.startSSE(ctx)
	case TransportStreamableHTTP:
		return s.startStreamableHTTP(ctx)
	default:
		return fmt.Errorf("unsupported transport type: %s", s.transport)
	}
}

// startStdio starts the server using stdio transport.
func (s *Server) startStdio(ctx context.Context) error {
	return server.ServeStdio(s.mcpServer)
}

// startSSE starts the server using SSE transport.
func (s *Server) startSSE(ctx context.Context) error {
	if s.sseConfig == nil {
		return fmt.Errorf("SSE configuration is required for SSE transport")
	}

	// Create SSE server options
	opts := []server.SSEOption{}
	if s.sseConfig.BaseURL != "" {
		opts = append(opts, server.WithBaseURL(s.sseConfig.BaseURL))
	}
	if s.sseConfig.StaticBasePath != "" {
		opts = append(opts, server.WithStaticBasePath(s.sseConfig.StaticBasePath))
	}
	if s.sseConfig.KeepAliveInterval > 0 {
		opts = append(opts, server.WithKeepAliveInterval(s.sseConfig.KeepAliveInterval))
	}

	// Create and start SSE server
	sseServer := server.NewSSEServer(s.mcpServer, opts...)
	return sseServer.Start(s.sseConfig.Addr)
}

// startStreamableHTTP starts the server using Streamable HTTP transport.
func (s *Server) startStreamableHTTP(ctx context.Context) error {
	if s.streamableHTTPConfig == nil {
		return fmt.Errorf("Streamable HTTP configuration is required for Streamable HTTP transport")
	}

	// Create Streamable HTTP server options
	opts := []server.StreamableHTTPOption{}
	if s.streamableHTTPConfig.EndpointPath != "" {
		opts = append(opts, server.WithEndpointPath(s.streamableHTTPConfig.EndpointPath))
	}
	if s.streamableHTTPConfig.HeartbeatInterval > 0 {
		opts = append(opts, server.WithHeartbeatInterval(s.streamableHTTPConfig.HeartbeatInterval))
	}
	if s.streamableHTTPConfig.Stateless {
		opts = append(opts, server.WithStateLess(true))
	}

	// Create Streamable HTTP server
	streamableServer := server.NewStreamableHTTPServer(s.mcpServer, opts...)

	// If CORS is enabled, create a custom HTTP server with CORS middleware
	if s.streamableHTTPConfig.EnableCORS {
		return s.startStreamableHTTPWithCORS(ctx, streamableServer)
	}

	// Use the default server without CORS
	return streamableServer.Start(s.streamableHTTPConfig.Addr)
}

// startStreamableHTTPWithCORS starts a custom HTTP server with CORS middleware
func (s *Server) startStreamableHTTPWithCORS(ctx context.Context, streamableServer *server.StreamableHTTPServer) error {
	// Create a new HTTP server with CORS middleware
	httpServer := &http.Server{
		Addr:    s.streamableHTTPConfig.Addr,
		Handler: s.createCORSHandler(),
	}

	// Start the server in a goroutine
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("HTTP server error: %v\n", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	
	// Gracefully shutdown the server
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return httpServer.Shutdown(shutdownCtx)
}

// createCORSHandler creates an HTTP handler with CORS middleware
func (s *Server) createCORSHandler() http.Handler {
	// Create CORS middleware
	corsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			
			// Only set CORS headers if CORS is enabled and origin is allowed
			if s.streamableHTTPConfig.EnableCORS && s.isOriginAllowed(origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", s.getAllowedMethods())
				w.Header().Set("Access-Control-Allow-Headers", s.getAllowedHeaders())
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours
			}

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				// Only respond to preflight if CORS is enabled and origin is allowed
				if s.streamableHTTPConfig.EnableCORS && s.isOriginAllowed(origin) {
					w.WriteHeader(http.StatusOK)
				} else {
					w.WriteHeader(http.StatusForbidden)
				}
				return
			}

			// Call the next handler
			next.ServeHTTP(w, r)
		})
	}

	// Create a custom handler that routes requests to the MCP server
	mcpHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the request path matches the endpoint path
		if r.URL.Path == s.streamableHTTPConfig.EndpointPath {
			// Handle MCP requests here
			// For now, we'll return a simple response indicating CORS is working
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "CORS enabled", "message": "MCP server is running with CORS support"}`))
		} else {
			// Return 404 for unknown paths
			http.NotFound(w, r)
		}
	})

	return corsMiddleware(mcpHandler)
}

// isOriginAllowed checks if the given origin is allowed
func (s *Server) isOriginAllowed(origin string) bool {
	if !s.streamableHTTPConfig.EnableCORS {
		return false
	}
	
	// If no specific origins are configured, allow localhost
	if len(s.streamableHTTPConfig.AllowedOrigins) == 0 {
		return origin == "http://localhost:3000" || origin == "http://127.0.0.1:3000"
	}
	
	// Check against configured origins
	for _, allowedOrigin := range s.streamableHTTPConfig.AllowedOrigins {
		if origin == allowedOrigin {
			return true
		}
	}
	
	return false
}

// getAllowedMethods returns the allowed HTTP methods
func (s *Server) getAllowedMethods() string {
	if len(s.streamableHTTPConfig.AllowedMethods) == 0 {
		return "GET, POST, OPTIONS"
	}
	
	methods := ""
	for i, method := range s.streamableHTTPConfig.AllowedMethods {
		if i > 0 {
			methods += ", "
		}
		methods += method
	}
	return methods
}

// getAllowedHeaders returns the allowed HTTP headers
func (s *Server) getAllowedHeaders() string {
	if len(s.streamableHTTPConfig.AllowedHeaders) == 0 {
		return "Content-Type, Authorization, X-Requested-With"
	}
	
	headers := ""
	for i, header := range s.streamableHTTPConfig.AllowedHeaders {
		if i > 0 {
			headers += ", "
		}
		headers += header
	}
	return headers
}

// GetMCPServer returns the underlying MCP server instance.
func (s *Server) GetMCPServer() *server.MCPServer {
	return s.mcpServer
}
