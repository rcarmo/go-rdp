package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"
	"time"

	"github.com/rcarmo/rdp-html5/internal/config"
	"github.com/rcarmo/rdp-html5/internal/handler"
	"github.com/rcarmo/rdp-html5/internal/logging"
)

const (
	appName    = "RDP HTML5 Client"
	appVersion = "1.0.0"
)

func main() {
	args, action := parseFlags()
	if action != "" {
		return
	}
	if err := run(args); err != nil {
		log.Fatalln(err)
	}
}

// parsedArgs holds the parsed command line arguments
type parsedArgs struct {
	host          string
	port          string
	logLevel      string
	skipTLS       bool
	allowAnyTLS   bool
	tlsServerName string
	useNLA        bool
	enableRFX     *bool // nil = use default, non-nil = override
}

// parseFlags parses command line flags and returns the parsed args.
// Returns action string if help/version was shown (caller should return early).
func parseFlags() (parsedArgs, string) {
	return parseFlagsWithArgs(os.Args[1:])
}

// parseFlagsWithArgs parses the given arguments and returns the parsed args.
func parseFlagsWithArgs(args []string) (parsedArgs, string) {
	fs := flag.NewFlagSet("server", flag.ContinueOnError)
	hostFlag := fs.String("host", "", "RDP HTML5 server host")
	portFlag := fs.String("port", "", "RDP HTML5 server port")
	logLevelFlag := fs.String("log-level", "", "log level (debug, info, warn, error)")
	skipTLS := fs.Bool("skip-tls-verify", false, "skip TLS certificate validation")
	allowAnyTLS := fs.Bool("tls-allow-any-server-name", false, "allow overriding server name to any host (disables SNI enforcement)")
	tlsServerName := fs.String("tls-server-name", "", "override TLS server name")
	useNLA := fs.Bool("nla", false, "enable Network Level Authentication (NLA/CredSSP)")
	noRFX := fs.Bool("no-rfx", false, "disable RemoteFX codec support")
	helpFlag := fs.Bool("help", false, "show help")
	versionFlag := fs.Bool("version", false, "show version")

	_ = fs.Parse(args)

	if *helpFlag {
		showHelp()
		return parsedArgs{}, "help"
	}

	if *versionFlag {
		showVersion()
		return parsedArgs{}, "version"
	}

	// Handle RFX flag - only set if explicitly disabled
	var enableRFX *bool
	if *noRFX {
		rfxValue := false
		enableRFX = &rfxValue
	}

	return parsedArgs{
		host:          strings.TrimSpace(*hostFlag),
		port:          strings.TrimSpace(*portFlag),
		logLevel:      strings.TrimSpace(*logLevelFlag),
		skipTLS:       *skipTLS,
		allowAnyTLS:   *allowAnyTLS,
		tlsServerName: strings.TrimSpace(*tlsServerName),
		useNLA:        *useNLA,
		enableRFX:     enableRFX,
	}, ""
}

// run starts the server with the given arguments
func run(args parsedArgs) error {
	opts := config.LoadOptions{
		Host:              args.host,
		Port:              args.port,
		LogLevel:          args.logLevel,
		SkipTLSValidation: args.skipTLS,
		AllowAnyTLSServer: args.allowAnyTLS,
		TLSServerName:     args.tlsServerName,
		UseNLA:            args.useNLA,
		EnableRFX:         args.enableRFX,
	}

	cfg, err := config.LoadWithOverrides(opts)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	setupLogging(cfg.Logging)

	server := createServer(cfg)
	rfxStatus := "enabled"
	if !cfg.RDP.EnableRFX {
		rfxStatus = "disabled"
	}
	logging.Info("Starting server on %s:%s (TLS=%t, RFX=%s)", cfg.Server.Host, cfg.Server.Port, cfg.Security.EnableTLS, rfxStatus)

	if err := startServer(server, cfg); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func createServer(cfg *config.Config) *http.Server {
	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir("./web")))
	mux.HandleFunc("/connect", handler.Connect)

	h := applySecurityMiddleware(mux, cfg)
	h = requestLoggingMiddleware(h)

	return &http.Server{
		Addr:         addr,
		Handler:      h,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}
}

func applySecurityMiddleware(next http.Handler, cfg *config.Config) http.Handler {
	if cfg == nil {
		return securityHeadersMiddleware(corsMiddleware(next, nil))
	}

	h := next
	if cfg.Security.EnableRateLimit {
		h = rateLimitMiddleware(h, cfg.Security.RateLimitPerMinute)
	}
	h = corsMiddleware(h, cfg.Security.AllowedOrigins)
	h = securityHeadersMiddleware(h)

	return h
}

func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		// Allow inline scripts/styles and WASM for the single-page UI
		// Allow data: URIs for img-src to support custom cursor images
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'wasm-unsafe-eval'; style-src 'self' 'unsafe-inline'; connect-src 'self' ws: wss:; img-src 'self' data:")

		next.ServeHTTP(w, r)
	})
}

func corsMiddleware(next http.Handler, allowedOrigins []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if isOriginAllowed(origin, allowedOrigins, r.Host) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isOriginAllowed(origin string, allowedOrigins []string, host string) bool {
	if origin == "" {
		return false
	}

	for _, allowed := range allowedOrigins {
		if strings.TrimSpace(allowed) == origin {
			return true
		}
	}

	// When no ALLOWED_ORIGINS configured, allow all origins (development mode)
	// For production, explicitly set ALLOWED_ORIGINS
	if len(allowedOrigins) == 0 {
		return true
	}

	return false
}

func rateLimitMiddleware(next http.Handler, _ int) http.Handler {
	// Simplified placeholder: production implementation should enforce rate limits
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

func setupLogging(cfg config.LoggingConfig) {
	log.SetFlags(log.LstdFlags | log.LUTC)
	log.SetOutput(log.Writer())
	
	// Set leveled logging level - default to "info" if not configured
	level := cfg.Level
	if level == "" {
		level = "info"
	}
	logging.SetLevelFromString(level)
}

func requestLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logging.Debug("%s %s %s %s", r.RemoteAddr, r.Method, r.URL.Path, time.Since(start))
	})
}

func startServer(server *http.Server, _ *config.Config) error {
	if server == nil {
		return fmt.Errorf("server is nil")
	}

	err := server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return err
}

func showHelp() {
	fmt.Println(appName)
	fmt.Println("USAGE: rdp-html5 [options]")
	fmt.Println("OPTIONS:")
	fmt.Println("  -host               Set server listen host (default 0.0.0.0)")
	fmt.Println("  -port               Set server listen port (default 8080)")
	fmt.Println("  -log-level          Set log level (debug, info, warn, error)")
	fmt.Println("  -skip-tls-verify    Skip TLS certificate validation")
	fmt.Println("  -tls-server-name    Override TLS server name")
	fmt.Println("  -nla                Enable Network Level Authentication")
	fmt.Println("  -no-rfx             Disable RemoteFX codec support")
	fmt.Println("  -version            Show version information")
	fmt.Println("  -help               Show this help message")
	fmt.Println("ENVIRONMENT VARIABLES: SERVER_HOST, SERVER_PORT, LOG_LEVEL, SKIP_TLS_VALIDATION, TLS_SERVER_NAME, RDP_ENABLE_RFX")
	fmt.Println("EXAMPLES: rdp-html5 -host 0.0.0.0 -port 8080")
}

func showVersion() {
	fmt.Printf("%s %s\n", appName, appVersion)
	fmt.Println("Built with Go", time.Now().Year())
	fmt.Println("Protocol: RDP 10.x")
}
