package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"strings"
	"time"

	"github.com/kulaginds/rdp-html5/internal/pkg/config"
	"github.com/kulaginds/rdp-html5/internal/pkg/handler"
)

const (
	appName    = "RDP HTML5 Client"
	appVersion = "v2.0.0"
)

func main() {
	hostFlag := flag.String("host", "", "RDP HTML5 server host")
	portFlag := flag.String("port", "", "RDP HTML5 server port")
	logLevelFlag := flag.String("log-level", "", "log level (debug, info, warn, error)")
	skipTLS := flag.Bool("skip-tls-verify", false, "skip TLS certificate validation")
	skipSSL := flag.Bool("skip-ssl-verify", false, "skip TLS certificate validation (deprecated alias)")
	tlsServerName := flag.String("tls-server-name", "", "override TLS server name")
	useNLA := flag.Bool("nla", false, "enable Network Level Authentication (NLA/CredSSP)")
	helpFlag := flag.Bool("help", false, "show help")
	versionFlag := flag.Bool("version", false, "show version")

	flag.Parse()

	if *helpFlag {
		showHelp()
		return
	}

	if *versionFlag {
		showVersion()
		return
	}

	opts := config.LoadOptions{
		Host:              strings.TrimSpace(*hostFlag),
		Port:              strings.TrimSpace(*portFlag),
		LogLevel:          strings.TrimSpace(*logLevelFlag),
		SkipTLSValidation: *skipTLS || *skipSSL,
		TLSServerName:     strings.TrimSpace(*tlsServerName),
		UseNLA:            *useNLA,
	}

	cfg, err := config.LoadWithOverrides(opts)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	setupLogging(cfg.Logging)

	server := createServer(cfg)
	log.Printf("starting server on %s:%s (TLS=%t)", cfg.Server.Host, cfg.Server.Port, cfg.Security.EnableTLS)

	if err := startServer(server, cfg); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalln(err)
	}
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
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'wasm-unsafe-eval'; style-src 'self' 'unsafe-inline'; connect-src 'self' ws: wss:")

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

	if len(allowedOrigins) == 0 {
		return strings.Contains(origin, host)
	}

	return false
}

func rateLimitMiddleware(next http.Handler, _ int) http.Handler {
	// Simplified placeholder: production implementation should enforce rate limits
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

func setupLogging(_ config.LoggingConfig) {
	log.SetFlags(log.LstdFlags | log.LUTC)
	log.SetOutput(log.Writer())
}

func requestLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s %s", r.RemoteAddr, r.Method, r.URL.Path, time.Since(start))
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
	fmt.Println("  -skip-ssl-verify    Skip TLS certificate validation (alias)")
	fmt.Println("  -tls-server-name    Override TLS server name")
	fmt.Println("  -version            Show version information")
	fmt.Println("  -help               Show this help message")
	fmt.Println("ENVIRONMENT VARIABLES: SERVER_HOST, SERVER_PORT, LOG_LEVEL, SKIP_TLS_VALIDATION, TLS_SERVER_NAME")
	fmt.Println("EXAMPLES: rdp-html5 -host 0.0.0.0 -port 8080")
}

func showVersion() {
	fmt.Printf("%s %s\n", appName, appVersion)
	fmt.Println("Built with Go", time.Now().Year())
	fmt.Println("Protocol: RDP 10.x")
}
