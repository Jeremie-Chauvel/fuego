package op

import (
	"html/template"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/golang-jwt/jwt/v5"
)

var isGo1_22 = strings.TrimPrefix(runtime.Version(), "devel ") >= "go1.22"

type OpenapiConfig struct {
	DisableSwagger    bool
	DisableLocalSave  bool
	SwaggerUrl        string
	JsonSpecUrl       string
	JsonSpecLocalPath string
}

var defaultOpenapiConfig = OpenapiConfig{
	SwaggerUrl:        "/swagger",
	JsonSpecUrl:       "/swagger/openapi.json",
	JsonSpecLocalPath: "doc/openapi.json",
}

type Server struct {
	middlewares []func(http.Handler) http.Handler
	mux         *http.ServeMux
	basePath    string

	spec openapi3.T

	Security Security

	autoAuth AutoAuthConfig
	fs       fs.FS
	template *template.Template // TODO: use preparsed templates

	Addr                  string
	DisallowUnknownFields bool // If true, the server will return an error if the request body contains unknown fields. Useful for quick debugging in development.
	maxBodySize           int64
	Serialize             func(w http.ResponseWriter, ans any)   // Used to serialize the response. Defaults to [SendJSON].
	SerializeError        func(w http.ResponseWriter, err error) // Used to serialize the error response. Defaults to [SendJSONError].
	ErrorHandler          func(err error) error                  // Used to transform any error into a unified error type structure with status code. Defaults to [ErrorHandler]
	startTime             time.Time

	OpenapiConfig OpenapiConfig
}

// NewServer creates a new server with the given options.
// For example:
//
//	app := op.NewServer(
//		op.WithPort(":8080"),
//		op.WithoutLogger(),
//	)
//
// Option all begin with `With`.
// Some default options are set in the function body.
func NewServer(options ...func(*Server)) *Server {
	s := &Server{
		mux:  http.NewServeMux(),
		spec: NewOpenAPI(),

		OpenapiConfig: defaultOpenapiConfig,

		Security: NewSecurity(),
	}

	defaultOptions := [...]func(*Server){
		WithPort(":8080"),
		WithDisallowUnknownFields(true),
		WithSerializer(SendJSON),
		WithErrorSerializer(SendJSONError),
		WithErrorHandler(ErrorHandler),
	}

	for _, option := range append(defaultOptions[:], options...) {
		option(s)
	}

	if !isGo1_22 {
		slog.Warn(
			"Please upgrade to Go >= 1.22. " +
				"You are running " + runtime.Version() + ": " +
				"you cannot use path params nor register routes with the same path but different methods. ")
	}

	s.startTime = time.Now()

	if s.autoAuth.Enabled {
		Post(s, "/auth/login", s.Security.LoginHandler(s.autoAuth.VerifyUserInfo)).SetTags("Auth").WithSummary("Login")
		PostStd(s, "/auth/logout", s.Security.CookieLogoutHandler).SetTags("Auth").WithSummary("Logout")

		s.middlewares = []func(http.Handler) http.Handler{
			s.Security.TokenToContext(TokenFromCookie, TokenFromHeader),
		}

		PostStd(s, "/auth/refresh", s.Security.RefreshHandler).SetTags("Auth").WithSummary("Refresh token")
	}

	return s
}

// WithTemplateFS sets the filesystem used to load templates.
// To be used with [WithTemplateGlobs] or [WithTemplates].
// For example:
//
//	WithTemplateFS(os.DirFS("./templates"))
//
// or with embedded templates:
//
//	//go:embed templates
//	var templates embed.FS
//	...
//	WithTemplateFS(templates)
func WithTemplateFS(fs fs.FS) func(*Server) {
	return func(c *Server) { c.fs = fs }
}

// WithTemplates loads the templates used to render HTML.
// To be used with [WithTemplateFS]. If not set, it will use the os filesystem, at folder "./templates".
func WithTemplates(templates *template.Template) func(*Server) {
	return func(s *Server) {
		if s.fs == nil {
			s.fs = os.DirFS("./templates")
			slog.Warn("No template filesystem set. Using os filesystem at './templates'.")
		}
		s.template = templates
	}
}

// WithTemplateGlobs loads templates matching the given patterns from the server filesystem.
// If the server filesystem is not set, it will use the os filesystem, at folder "./templates".
// For example:
//
//	WithTemplateGlobs("**/*.html")
func WithTemplateGlobs(patterns ...string) func(*Server) {
	return func(s *Server) {
		if s.fs == nil {
			s.fs = os.DirFS("./templates")
			slog.Warn("No template filesystem set. Using os filesystem at './templates'.")
		}
		err := s.loadTemplates(patterns...)
		if err != nil {
			slog.Error("Error loading templates", "error", err)
			panic(err)
		}
	}
}

func WithBasePath(basePath string) func(*Server) {
	return func(c *Server) { c.basePath = basePath }
}

func WithMaxBodySize(maxBodySize int64) func(*Server) {
	return func(c *Server) { c.maxBodySize = maxBodySize }
}

func WithAutoAuth(verifyUserInfo func(user, password string) (jwt.Claims, error)) func(*Server) {
	return func(c *Server) {
		c.autoAuth.Enabled = true
		c.autoAuth.VerifyUserInfo = verifyUserInfo
	}
}

// WithDisallowUnknownFields sets the DisallowUnknownFields option.
// If true, the server will return an error if the request body contains unknown fields.
// Useful for quick debugging in development.
// Defaults to true.
func WithDisallowUnknownFields(b bool) func(*Server) {
	return func(c *Server) { c.DisallowUnknownFields = b }
}

// WithPort sets the port of the server. For example, ":8080".
func WithPort(port string) func(*Server) {
	return func(c *Server) { c.Addr = port }
}

func WithXML() func(*Server) {
	return func(c *Server) {
		c.Serialize = SendXML
		c.SerializeError = SendXMLError
	}
}

func WithHandler(handler slog.Handler) func(*Server) {
	return func(c *Server) {
		if handler != nil {
			slog.SetDefault(slog.New(handler))
		}
	}
}

func WithSerializer(serializer func(w http.ResponseWriter, ans any)) func(*Server) {
	return func(c *Server) { c.Serialize = serializer }
}

func WithErrorSerializer(serializer func(w http.ResponseWriter, err error)) func(*Server) {
	return func(c *Server) { c.SerializeError = serializer }
}

func WithErrorHandler(errorHandler func(err error) error) func(*Server) {
	return func(c *Server) { c.ErrorHandler = errorHandler }
}

// WithoutLogger disables the default logger.
func WithoutLogger() func(*Server) {
	return func(c *Server) {
		slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	}
}

func WithOpenapiConfig(openapiConfig OpenapiConfig) func(*Server) {
	return func(s *Server) {
		s.OpenapiConfig = openapiConfig

		if s.OpenapiConfig.JsonSpecUrl == "" {
			s.OpenapiConfig.JsonSpecUrl = defaultOpenapiConfig.JsonSpecUrl
		}

		if s.OpenapiConfig.SwaggerUrl == "" {
			s.OpenapiConfig.SwaggerUrl = defaultOpenapiConfig.SwaggerUrl
		}

		if s.OpenapiConfig.JsonSpecLocalPath == "" {
			s.OpenapiConfig.JsonSpecLocalPath = defaultOpenapiConfig.JsonSpecLocalPath
		}

		if !validateJsonSpecLocalPath(s.OpenapiConfig.JsonSpecLocalPath) {
			slog.Error("Error writing json spec. Value of 'jsonSpecLocalPath' option is not valid", "file", s.OpenapiConfig.JsonSpecLocalPath)
			return
		}

		if !validateJsonSpecUrl(s.OpenapiConfig.JsonSpecUrl) {
			slog.Error("Error serving openapi json spec. Value of 's.OpenapiConfig.JsonSpecUrl' option is not valid", "url", s.OpenapiConfig.JsonSpecUrl)
			return
		}

		if !validateSwaggerUrl(s.OpenapiConfig.SwaggerUrl) {
			slog.Error("Error serving swagger ui. Value of 's.OpenapiConfig.SwaggerUrl' option is not valid", "url", s.OpenapiConfig.SwaggerUrl)
			return
		}
	}
}
