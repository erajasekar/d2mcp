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
	AllowedOrigins    []string
	AllowedMethods    []string
	AllowedHeaders    []string
	AllowCredentials  bool
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

// corsMiddleware creates a CORS middleware function
func corsMiddleware(config *StreamableHTTPConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set CORS headers
			if len(config.AllowedOrigins) > 0 {
				origin := r.Header.Get("Origin")
				for _, allowedOrigin := range config.AllowedOrigins {
					if allowedOrigin == "*" || allowedOrigin == origin {
						w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
						break
					}
				}
			} else {
				// Default: allow all origins
				w.Header().Set("Access-Control-Allow-Origin", "*")
			}

			if len(config.AllowedMethods) > 0 {
				methods := ""
				for i, method := range config.AllowedMethods {
					if i > 0 {
						methods += ", "
					}
					methods += method
				}
				w.Header().Set("Access-Control-Allow-Methods", methods)
			} else {
				// Default: allow common methods
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			}

			if len(config.AllowedHeaders) > 0 {
				headers := ""
				for i, header := range config.AllowedHeaders {
					if i > 0 {
						headers += ", "
					}
					headers += header
				}
				w.Header().Set("Access-Control-Allow-Headers", headers)
			} else {
				// Default: allow common headers and MCP-specific headers
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, mcp-protocol-version, mcp-session-id")
			}

			if config.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// healthHandler provides a simple health check endpoint
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy","service":"d2mcp"}`))
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
	
	// Create a custom mux to handle both MCP and health endpoints
	mux := http.NewServeMux()
	
	// Add health check endpoint
	mux.HandleFunc("/health", healthHandler)
	
	// Add MCP endpoint with CORS middleware
	mcpHandler := corsMiddleware(s.streamableHTTPConfig)(streamableServer)
	mux.Handle(s.streamableHTTPConfig.EndpointPath, mcpHandler)
	
	// Handle root path requests by redirecting to MCP endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, s.streamableHTTPConfig.EndpointPath, http.StatusFound)
			return
		}
		http.NotFound(w, r)
	})

	// Create HTTP server with our custom mux
	httpServer := &http.Server{
		Addr:    s.streamableHTTPConfig.Addr,
		Handler: mux,
	}

	return httpServer.ListenAndServe()
}

// GetMCPServer returns the underlying MCP server instance.
func (s *Server) GetMCPServer() *server.MCPServer {
	return s.mcpServer
}
