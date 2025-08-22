// Package web provides a comprehensive modern web framework for Orizon.
// This package includes advanced routing, middleware, templating, WebSocket support,
// GraphQL, REST APIs, microservices, server-sent events, and real-time features.
package web

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"path"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Router represents the main web router.
type Router struct {
	routes       []Route
	middleware   []MiddlewareFunc
	notFound     HandlerFunc
	errorHandler ErrorHandlerFunc
	templates    *template.Template
	mu           sync.RWMutex
}

// Route represents a single route.
type Route struct {
	Method     string
	Pattern    string
	Handler    HandlerFunc
	Middleware []MiddlewareFunc
	Regex      *regexp.Regexp
	ParamNames []string
}

// Context represents the request context.
type Context struct {
	Request  *http.Request
	Response *ResponseWriter
	params   map[string]string
	query    url.Values
	data     map[string]interface{}
	status   int
	router   *Router
}

// WebSocket represents a WebSocket connection.
type WebSocket struct {
	conn            *http.Response // Simplified WebSocket representation
	readBuffer      []byte
	writeBuffer     []byte
	isConnected     bool
	messageHandlers map[string]MessageHandler
	closeHandlers   []CloseHandler
	mutex           sync.Mutex
}

// MessageHandler handles WebSocket messages.
type MessageHandler func(*WebSocket, []byte)

// CloseHandler handles WebSocket close events.
type CloseHandler func(*WebSocket, int, string)

// GraphQLResolver represents a GraphQL resolver.
type GraphQLResolver struct {
	Schema     *GraphQLSchema
	Resolvers  map[string]ResolverFunc
	Middleware []GraphQLMiddleware
}

// GraphQLSchema represents a GraphQL schema.
type GraphQLSchema struct {
	Types         map[string]*GraphQLType
	Queries       map[string]*GraphQLField
	Mutations     map[string]*GraphQLField
	Subscriptions map[string]*GraphQLField
}

// GraphQLType represents a GraphQL type.
type GraphQLType struct {
	Name          string
	Kind          GraphQLKind
	Fields        map[string]*GraphQLField
	Interfaces    []*GraphQLType
	PossibleTypes []*GraphQLType
}

// GraphQLKind represents GraphQL type kinds.
type GraphQLKind int

const (
	ScalarType GraphQLKind = iota
	ObjectType
	InterfaceType
	UnionType
	EnumType
	InputObjectType
	ListType
	NonNullType
)

// GraphQLField represents a GraphQL field.
type GraphQLField struct {
	Name        string
	Type        *GraphQLType
	Args        map[string]*GraphQLArgument
	Resolver    ResolverFunc
	Description string
}

// GraphQLArgument represents a GraphQL argument.
type GraphQLArgument struct {
	Name         string
	Type         *GraphQLType
	DefaultValue interface{}
	Description  string
}

// ResolverFunc represents a GraphQL resolver function.
type ResolverFunc func(*GraphQLContext) (interface{}, error)

// GraphQLContext represents GraphQL execution context.
type GraphQLContext struct {
	Request    *GraphQLRequest
	Schema     *GraphQLSchema
	Variables  map[string]interface{}
	Context    context.Context
	FieldName  string
	ParentType *GraphQLType
	Source     interface{}
	Args       map[string]interface{}
}

// GraphQLRequest represents a GraphQL request.
type GraphQLRequest struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables"`
	OperationName string                 `json:"operationName"`
}

// GraphQLResponse represents a GraphQL response.
type GraphQLResponse struct {
	Data   interface{}    `json:"data"`
	Errors []GraphQLError `json:"errors,omitempty"`
}

// GraphQLError represents a GraphQL error.
type GraphQLError struct {
	Message    string                 `json:"message"`
	Locations  []GraphQLLocation      `json:"locations,omitempty"`
	Path       []interface{}          `json:"path,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

// GraphQLLocation represents a location in GraphQL query.
type GraphQLLocation struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// GraphQLMiddleware represents GraphQL middleware.
type GraphQLMiddleware func(*GraphQLContext, ResolverFunc) (interface{}, error)

// APIGateway represents an API gateway.
type APIGateway struct {
	Services     map[string]*Service
	LoadBalancer *LoadBalancer
	RateLimit    *RateLimiter
	Auth         *AuthProvider
	Logger       *Logger
	Health       *HealthChecker
	Metrics      *MetricsCollector
}

// Service represents a microservice.
type Service struct {
	Name        string
	Instances   []ServiceInstance
	HealthCheck HealthCheck
	Circuit     *CircuitBreaker
	Config      ServiceConfig
}

// ServiceInstance represents a service instance.
type ServiceInstance struct {
	ID       string
	Address  string
	Port     int
	Health   HealthStatus
	Metadata map[string]string
	Weight   int
	Tags     []string
}

// ServiceConfig represents service configuration.
type ServiceConfig struct {
	Timeout          time.Duration
	RetryCount       int
	RetryDelay       time.Duration
	LoadBalancing    LoadBalancingStrategy
	FailureThreshold int
}

// LoadBalancer represents a load balancer.
type LoadBalancer struct {
	Strategy LoadBalancingStrategy
	Services map[string][]ServiceInstance
	Health   *HealthChecker
}

// LoadBalancingStrategy represents load balancing strategies.
type LoadBalancingStrategy int

const (
	RoundRobin LoadBalancingStrategy = iota
	WeightedRoundRobin
	LeastConnections
	Random
	IPHash
	ConsistentHash
)

// CircuitBreaker represents a circuit breaker pattern.
type CircuitBreaker struct {
	State         CircuitState
	FailureCount  int
	SuccessCount  int
	Threshold     int
	Timeout       time.Duration
	LastFailure   time.Time
	OnStateChange func(CircuitState)
	mutex         sync.Mutex
}

// CircuitState represents circuit breaker states.
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// RateLimiter represents a rate limiter.
type RateLimiter struct {
	Limits   map[string]*RateLimit
	Storage  RateLimitStorage
	Strategy RateLimitStrategy
}

// RateLimit represents rate limit configuration.
type RateLimit struct {
	Requests int
	Window   time.Duration
	Burst    int
}

// RateLimitStorage represents rate limit storage interface.
type RateLimitStorage interface {
	Get(key string) (int, time.Time, error)
	Set(key string, value int, expiry time.Time) error
	Increment(key string, expiry time.Time) (int, error)
}

// RateLimitStrategy represents rate limiting strategies.
type RateLimitStrategy int

const (
	TokenBucket RateLimitStrategy = iota
	LeakyBucket
	FixedWindow
	SlidingWindow
)

// AuthProvider represents authentication provider.
type AuthProvider struct {
	JWT        *JWTConfig
	OAuth2     *OAuth2Config
	Sessions   SessionStore
	Validators map[string]AuthValidator
}

// JWTConfig represents JWT configuration.
type JWTConfig struct {
	Secret     string
	Algorithm  string
	Expiration time.Duration
	Issuer     string
	Audience   string
}

// OAuth2Config represents OAuth2 configuration.
type OAuth2Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
	AuthURL      string
	TokenURL     string
}

// SessionStore represents session storage interface.
type SessionStore interface {
	Get(sessionID string) (*Session, error)
	Set(session *Session) error
	Delete(sessionID string) error
	Cleanup() error
}

// Session represents a user session.
type Session struct {
	ID        string
	Data      map[string]interface{}
	CreatedAt time.Time
	ExpiresAt time.Time
	UserID    string
}

// AuthValidator represents an authentication validator.
type AuthValidator func(*Context) (bool, error)

// Logger represents a structured logger.
type Logger struct {
	Level  LogLevel
	Output io.Writer
	Format LogFormat
	Fields map[string]interface{}
	Hooks  []LogHook
}

// LogLevel represents log levels.
type LogLevel int

const (
	LogTrace LogLevel = iota
	LogDebug
	LogInfo
	LogWarn
	LogError
	LogFatal
)

// LogFormat represents log formats.
type LogFormat int

const (
	JSONFormat LogFormat = iota
	TextFormat
	StructuredFormat
)

// LogHook represents a log hook.
type LogHook func(*LogEntry)

// LogEntry represents a log entry.
type LogEntry struct {
	Level     LogLevel
	Message   string
	Timestamp time.Time
	Fields    map[string]interface{}
	Error     error
}

// HealthChecker represents a health checker.
type HealthChecker struct {
	Checks   map[string]HealthCheck
	Interval time.Duration
	Timeout  time.Duration
	Results  map[string]*HealthResult
	mutex    sync.RWMutex
}

// HealthCheck represents a health check function.
type HealthCheck func() error

// HealthStatus represents health status.
type HealthStatus int

const (
	HealthUnknown HealthStatus = iota
	HealthHealthy
	HealthUnhealthy
	HealthWarning
)

// HealthResult represents a health check result.
type HealthResult struct {
	Status    HealthStatus
	Message   string
	Timestamp time.Time
	Duration  time.Duration
	Error     error
}

// MetricsCollector represents a metrics collector.
type MetricsCollector struct {
	Counters   map[string]*Counter
	Gauges     map[string]*Gauge
	Histograms map[string]*Histogram
	Timers     map[string]*Timer
	Registry   MetricsRegistry
}

// Counter represents a counter metric.
type Counter struct {
	Value int64
	mutex sync.Mutex
}

// Gauge represents a gauge metric.
type Gauge struct {
	Value float64
	mutex sync.Mutex
}

// Histogram represents a histogram metric.
type Histogram struct {
	Buckets []HistogramBucket
	Count   int64
	Sum     float64
	mutex   sync.Mutex
}

// HistogramBucket represents a histogram bucket.
type HistogramBucket struct {
	UpperBound float64
	Count      int64
}

// Timer represents a timer metric.
type Timer struct {
	Count int64
	Sum   time.Duration
	Min   time.Duration
	Max   time.Duration
	mutex sync.Mutex
}

// MetricsRegistry represents a metrics registry.
type MetricsRegistry interface {
	Register(name string, metric interface{}) error
	Unregister(name string) error
	Get(name string) interface{}
	Export() map[string]interface{}
}

// WebSocketHub represents a WebSocket hub for managing connections.
type WebSocketHub struct {
	Connections map[string]*WebSocket
	Broadcast   chan []byte
	Register    chan *WebSocket
	Unregister  chan *WebSocket
	Rooms       map[string]*Room
	mutex       sync.RWMutex
}

// Room represents a WebSocket room.
type Room struct {
	ID          string
	Connections map[string]*WebSocket
	Broadcast   chan []byte
	mutex       sync.RWMutex
}

// SSEClient represents a Server-Sent Events client.
type SSEClient struct {
	ID          string
	Events      chan SSEEvent
	Done        chan struct{}
	Writer      http.ResponseWriter
	Flusher     http.Flusher
	LastEventID string
}

// SSEEvent represents a Server-Sent Events event.
type SSEEvent struct {
	ID    string
	Event string
	Data  string
	Retry int
}

// SSEBroker represents a Server-Sent Events broker.
type SSEBroker struct {
	Clients        map[string]*SSEClient
	NewClients     chan *SSEClient
	ClosingClients chan *SSEClient
	Events         chan SSEEvent
	mutex          sync.RWMutex
}

// Cache represents a caching layer.
type Cache struct {
	Store    CacheStore
	TTL      time.Duration
	MaxSize  int64
	Strategy CacheStrategy
}

// CacheStore represents a cache storage interface.
type CacheStore interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, ttl time.Duration) error
	Delete(key string) error
	Clear() error
	Size() int64
}

// CacheStrategy represents caching strategies.
type CacheStrategy int

const (
	LRU CacheStrategy = iota
	LFU
	FIFO
	Random
)

// Template represents template management.
type Template struct {
	Templates map[string]*template.Template
	Functions template.FuncMap
	Globals   map[string]interface{}
	Cache     bool
	Debug     bool
}

// StaticFileServer represents static file serving.
type StaticFileServer struct {
	Root         string
	IndexFile    string
	EnableBrowse bool
	Cache        *Cache
	Compress     bool
	ETags        bool
}

// Compression represents response compression.
type Compression struct {
	Level     int
	MinSize   int
	Types     []string
	Encodings []string
}

// CORS represents Cross-Origin Resource Sharing configuration.
type CORS struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

// Security represents security configuration.
type Security struct {
	HTTPS              bool
	TLSConfig          *tls.Config
	HSTS               *HSTS
	CSP                *CSP
	XFrameOptions      string
	XSSProtection      bool
	ContentTypeNoSniff bool
}

// HSTS represents HTTP Strict Transport Security.
type HSTS struct {
	MaxAge            int
	IncludeSubdomains bool
	Preload           bool
}

// CSP represents Content Security Policy.
type CSP struct {
	DefaultSrc []string
	ScriptSrc  []string
	StyleSrc   []string
	ImgSrc     []string
	ConnectSrc []string
	FontSrc    []string
	ObjectSrc  []string
	MediaSrc   []string
	FrameSrc   []string
	ReportURI  string
	ReportOnly bool
}

// ResponseWriter wraps http.ResponseWriter with additional functionality.
type ResponseWriter struct {
	http.ResponseWriter
	size   int
	status int
}

// HandlerFunc represents a request handler function.
type HandlerFunc func(*Context) error

// MiddlewareFunc represents a middleware function.
type MiddlewareFunc func(HandlerFunc) HandlerFunc

// ErrorHandlerFunc represents an error handler function.
type ErrorHandlerFunc func(*Context, error)

// NewRouter creates a new router instance.
func NewRouter() *Router {
	return &Router{
		routes:     make([]Route, 0),
		middleware: make([]MiddlewareFunc, 0),
		notFound: func(c *Context) error {
			return c.Status(404).Text("Not Found")
		},
		errorHandler: func(c *Context, err error) {
			c.Status(500).Text("Internal Server Error: " + err.Error())
		},
	}
}

// HTTP method helpers
func (r *Router) GET(pattern string, handler HandlerFunc) *Route {
	return r.Add("GET", pattern, handler)
}

func (r *Router) POST(pattern string, handler HandlerFunc) *Route {
	return r.Add("POST", pattern, handler)
}

func (r *Router) PUT(pattern string, handler HandlerFunc) *Route {
	return r.Add("PUT", pattern, handler)
}

func (r *Router) DELETE(pattern string, handler HandlerFunc) *Route {
	return r.Add("DELETE", pattern, handler)
}

func (r *Router) PATCH(pattern string, handler HandlerFunc) *Route {
	return r.Add("PATCH", pattern, handler)
}

func (r *Router) OPTIONS(pattern string, handler HandlerFunc) *Route {
	return r.Add("OPTIONS", pattern, handler)
}

func (r *Router) HEAD(pattern string, handler HandlerFunc) *Route {
	return r.Add("HEAD", pattern, handler)
}

// Add adds a new route to the router.
func (r *Router) Add(method, pattern string, handler HandlerFunc) *Route {
	r.mu.Lock()
	defer r.mu.Unlock()

	route := Route{
		Method:     method,
		Pattern:    pattern,
		Handler:    handler,
		Middleware: make([]MiddlewareFunc, 0),
	}

	// Compile regex for route matching
	route.Regex, route.ParamNames = compileRoute(pattern)

	r.routes = append(r.routes, route)
	return &r.routes[len(r.routes)-1]
}

// Use adds middleware to the router.
func (r *Router) Use(middleware ...MiddlewareFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.middleware = append(r.middleware, middleware...)
}

// Group creates a route group with common prefix and middleware.
func (r *Router) Group(prefix string, middleware ...MiddlewareFunc) *Group {
	return &Group{
		router:     r,
		prefix:     prefix,
		middleware: middleware,
	}
}

// Static serves static files from a directory.
func (r *Router) Static(prefix, root string) {
	fileServer := http.FileServer(http.Dir(root))
	r.GET(prefix+"/*filepath", func(c *Context) error {
		filepath := c.Param("filepath")
		c.Request.URL.Path = filepath
		fileServer.ServeHTTP(c.Response.ResponseWriter, c.Request)
		return nil
	})
}

// LoadTemplates loads HTML templates.
func (r *Router) LoadTemplates(pattern string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	tmpl, err := template.ParseGlob(pattern)
	if err != nil {
		return err
	}

	r.templates = tmpl
	return nil
}

// SetNotFound sets the not found handler.
func (r *Router) SetNotFound(handler HandlerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.notFound = handler
}

// SetErrorHandler sets the error handler.
func (r *Router) SetErrorHandler(handler ErrorHandlerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.errorHandler = handler
}

// ServeHTTP implements http.Handler interface.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx := &Context{
		Request:  req,
		Response: &ResponseWriter{ResponseWriter: w, status: 200},
		params:   make(map[string]string),
		query:    req.URL.Query(),
		data:     make(map[string]interface{}),
		router:   r,
	}

	// Find matching route
	route, found := r.findRoute(req.Method, req.URL.Path, ctx)

	var handler HandlerFunc
	if found {
		handler = route.Handler

		// Apply route-specific middleware
		for i := len(route.Middleware) - 1; i >= 0; i-- {
			handler = route.Middleware[i](handler)
		}
	} else {
		handler = r.notFound
	}

	// Apply global middleware
	for i := len(r.middleware) - 1; i >= 0; i-- {
		handler = r.middleware[i](handler)
	}

	// Execute handler
	if err := handler(ctx); err != nil {
		r.errorHandler(ctx, err)
	}
}

func (r *Router) findRoute(method, path string, ctx *Context) (*Route, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, route := range r.routes {
		if route.Method == method && route.Regex.MatchString(path) {
			// Extract parameters
			matches := route.Regex.FindStringSubmatch(path)
			for i, name := range route.ParamNames {
				if i+1 < len(matches) {
					ctx.params[name] = matches[i+1]
				}
			}
			return &route, true
		}
	}

	return nil, false
}

func compileRoute(pattern string) (*regexp.Regexp, []string) {
	var paramNames []string
	regexPattern := "^"

	parts := strings.Split(pattern, "/")
	for _, part := range parts {
		if part == "" {
			continue
		}

		regexPattern += "/"

		if strings.HasPrefix(part, ":") {
			// Named parameter
			paramName := part[1:]
			paramNames = append(paramNames, paramName)
			regexPattern += "([^/]+)"
		} else if strings.HasPrefix(part, "*") {
			// Wildcard parameter
			paramName := part[1:]
			paramNames = append(paramNames, paramName)
			regexPattern += "(.*)"
		} else {
			// Literal part
			regexPattern += regexp.QuoteMeta(part)
		}
	}

	regexPattern += "$"

	regex := regexp.MustCompile(regexPattern)
	return regex, paramNames
}

// Group represents a route group.
type Group struct {
	router     *Router
	prefix     string
	middleware []MiddlewareFunc
}

// GET adds a GET route to the group.
func (g *Group) GET(pattern string, handler HandlerFunc) *Route {
	return g.Add("GET", pattern, handler)
}

// POST adds a POST route to the group.
func (g *Group) POST(pattern string, handler HandlerFunc) *Route {
	return g.Add("POST", pattern, handler)
}

// PUT adds a PUT route to the group.
func (g *Group) PUT(pattern string, handler HandlerFunc) *Route {
	return g.Add("PUT", pattern, handler)
}

// DELETE adds a DELETE route to the group.
func (g *Group) DELETE(pattern string, handler HandlerFunc) *Route {
	return g.Add("DELETE", pattern, handler)
}

// Add adds a route to the group.
func (g *Group) Add(method, pattern string, handler HandlerFunc) *Route {
	fullPattern := path.Join(g.prefix, pattern)
	route := g.router.Add(method, fullPattern, handler)

	// Add group middleware
	route.Middleware = append(route.Middleware, g.middleware...)

	return route
}

// Use adds middleware to the group.
func (g *Group) Use(middleware ...MiddlewareFunc) {
	g.middleware = append(g.middleware, middleware...)
}

// Context methods

// Param returns a route parameter value.
func (c *Context) Param(name string) string {
	return c.params[name]
}

// Query returns a query parameter value.
func (c *Context) Query(name string) string {
	return c.query.Get(name)
}

// QueryDefault returns a query parameter value with a default.
func (c *Context) QueryDefault(name, defaultValue string) string {
	value := c.query.Get(name)
	if value == "" {
		return defaultValue
	}
	return value
}

// Set stores a value in the context.
func (c *Context) Set(key string, value interface{}) {
	c.data[key] = value
}

// Get retrieves a value from the context.
func (c *Context) Get(key string) interface{} {
	return c.data[key]
}

// GetString retrieves a string value from the context.
func (c *Context) GetString(key string) string {
	if value, ok := c.data[key].(string); ok {
		return value
	}
	return ""
}

// GetInt retrieves an int value from the context.
func (c *Context) GetInt(key string) int {
	if value, ok := c.data[key].(int); ok {
		return value
	}
	return 0
}

// Status sets the HTTP status code.
func (c *Context) Status(code int) *Context {
	c.status = code
	c.Response.WriteHeader(code)
	return c
}

// Header sets a response header.
func (c *Context) Header(key, value string) *Context {
	c.Response.Header().Set(key, value)
	return c
}

// JSON sends a JSON response.
func (c *Context) JSON(data interface{}) error {
	c.Header("Content-Type", "application/json")
	return json.NewEncoder(c.Response.ResponseWriter).Encode(data)
}

// Text sends a text response.
func (c *Context) Text(text string) error {
	c.Header("Content-Type", "text/plain")
	_, err := c.Response.Write([]byte(text))
	return err
}

// HTML renders an HTML template.
func (c *Context) HTML(templateName string, data interface{}) error {
	c.Header("Content-Type", "text/html")

	if c.router.templates == nil {
		return fmt.Errorf("no templates loaded")
	}

	return c.router.templates.ExecuteTemplate(c.Response.ResponseWriter, templateName, data)
}

// Redirect sends a redirect response.
func (c *Context) Redirect(code int, url string) error {
	c.Header("Location", url)
	c.Status(code)
	return nil
}

// Bind binds request data to a struct.
func (c *Context) Bind(obj interface{}) error {
	contentType := c.Request.Header.Get("Content-Type")

	if strings.Contains(contentType, "application/json") {
		return json.NewDecoder(c.Request.Body).Decode(obj)
	}

	if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		return c.bindForm(obj)
	}

	return fmt.Errorf("unsupported content type: %s", contentType)
}

func (c *Context) bindForm(obj interface{}) error {
	if err := c.Request.ParseForm(); err != nil {
		return err
	}

	v := reflect.ValueOf(obj).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		if !field.CanSet() {
			continue
		}

		tag := fieldType.Tag.Get("form")
		if tag == "" {
			tag = strings.ToLower(fieldType.Name)
		}

		values := c.Request.Form[tag]
		if len(values) == 0 {
			continue
		}

		value := values[0]

		switch field.Kind() {
		case reflect.String:
			field.SetString(value)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
				field.SetInt(intVal)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if uintVal, err := strconv.ParseUint(value, 10, 64); err == nil {
				field.SetUint(uintVal)
			}
		case reflect.Float32, reflect.Float64:
			if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
				field.SetFloat(floatVal)
			}
		case reflect.Bool:
			if boolVal, err := strconv.ParseBool(value); err == nil {
				field.SetBool(boolVal)
			}
		}
	}

	return nil
}

// ResponseWriter methods

// Write writes data to the response.
func (w *ResponseWriter) Write(data []byte) (int, error) {
	w.size += len(data)
	return w.ResponseWriter.Write(data)
}

// WriteHeader writes the status code.
func (w *ResponseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// Size returns the response size.
func (w *ResponseWriter) Size() int {
	return w.size
}

// Status returns the status code.
func (w *ResponseWriter) Status() int {
	return w.status
}

// Common middleware implementations

// Logger middleware logs requests.
func Logger() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			start := time.Now()

			err := next(c)

			duration := time.Since(start)
			fmt.Printf("[%s] %s %s - %d - %v\n",
				start.Format("2006/01/02 15:04:05"),
				c.Request.Method,
				c.Request.URL.Path,
				c.Response.Status(),
				duration,
			)

			return err
		}
	}
}

// CORS middleware handles Cross-Origin Resource Sharing.
func CORS() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			c.Header("Access-Control-Allow-Origin", "*")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if c.Request.Method == "OPTIONS" {
				return c.Status(204).Text("")
			}

			return next(c)
		}
	}
}

// Recovery middleware recovers from panics.
func Recovery() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("panic recovered: %v", r)
				}
			}()

			return next(c)
		}
	}
}

// BasicAuth middleware provides basic authentication.
func BasicAuth(username, password string) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			user, pass, ok := c.Request.BasicAuth()
			if !ok || user != username || pass != password {
				c.Header("WWW-Authenticate", "Basic realm=\"Restricted\"")
				return c.Status(401).Text("Unauthorized")
			}

			return next(c)
		}
	}
}

// RateLimit middleware implements rate limiting.
func RateLimit(requestsPerMinute int) MiddlewareFunc {
	type client struct {
		requests []time.Time
		mu       sync.Mutex
	}

	clients := make(map[string]*client)
	mu := sync.RWMutex{}

	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			clientIP := c.Request.RemoteAddr

			mu.RLock()
			cli, exists := clients[clientIP]
			mu.RUnlock()

			if !exists {
				cli = &client{
					requests: make([]time.Time, 0),
				}
				mu.Lock()
				clients[clientIP] = cli
				mu.Unlock()
			}

			cli.mu.Lock()
			now := time.Now()
			cutoff := now.Add(-time.Minute)

			// Remove old requests
			i := 0
			for _, reqTime := range cli.requests {
				if reqTime.After(cutoff) {
					cli.requests[i] = reqTime
					i++
				}
			}
			cli.requests = cli.requests[:i]

			// Check rate limit
			if len(cli.requests) >= requestsPerMinute {
				cli.mu.Unlock()
				return c.Status(429).Text("Rate limit exceeded")
			}

			// Add current request
			cli.requests = append(cli.requests, now)
			cli.mu.Unlock()

			return next(c)
		}
	}
}

// Server represents an HTTP server.
type Server struct {
	router   *Router
	server   *http.Server
	addr     string
	certFile string
	keyFile  string
}

// NewServer creates a new HTTP server.
func NewServer(addr string, router *Router) *Server {
	return &Server{
		router: router,
		addr:   addr,
		server: &http.Server{
			Addr:    addr,
			Handler: router,
		},
	}
}

// WithTLS configures the server for HTTPS.
func (s *Server) WithTLS(certFile, keyFile string) *Server {
	s.certFile = certFile
	s.keyFile = keyFile
	return s
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	fmt.Printf("Starting server on %s\n", s.addr)

	if s.certFile != "" && s.keyFile != "" {
		return s.server.ListenAndServeTLS(s.certFile, s.keyFile)
	}

	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
