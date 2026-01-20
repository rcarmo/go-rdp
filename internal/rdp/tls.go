package rdp

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/rcarmo/rdp-html5/internal/config"
)

func (c *Client) StartTLS() error {
	// Use client-specific TLS configuration if set, otherwise fall back to global config
	insecureSkipVerify := c.skipTLSValidation
	serverName := c.tlsServerName
	minTLSVersion := "1.2" // default

	// Pull server-wide config when available so flag/env overrides are honored
	cfg := config.GetGlobalConfig()
	if cfg == nil {
		var err error
		cfg, err = config.Load()
		if err != nil {
			cfg = &config.Config{}
		}
	}

	if cfg != nil {
		if !insecureSkipVerify {
			insecureSkipVerify = cfg.Security.SkipTLSValidation
		}
		if serverName == "" {
			serverName = cfg.Security.TLSServerName
		}
		if cfg.Security.MinTLSVersion != "" {
			minTLSVersion = cfg.Security.MinTLSVersion
		}
	}

	if serverName == "" {
		serverName = c.getServerName()
	}

	// When explicitly skipping verification, allow legacy TLS for compatibility with older servers
	if insecureSkipVerify {
		minTLSVersion = "1.0"
	}

	// Create TLS configuration with improved error handling
	tlsConfig := &tls.Config{
		InsecureSkipVerify: insecureSkipVerify,
		MinVersion:         c.getMinTLSVersion(minTLSVersion),
		MaxVersion:         tls.VersionTLS13,
		ServerName:         serverName,
	}

	// When skipping TLS validation, ensure we have a ServerName
	// If no ServerName is provided and we're skipping validation, use a fallback
	if tlsConfig.InsecureSkipVerify && tlsConfig.ServerName == "" {
		// Try to extract hostname from the connection address
		if c.conn != nil {
			remoteAddr := c.conn.RemoteAddr().String()
			host, _, err := net.SplitHostPort(remoteAddr)
			if err == nil && host != "" {
				tlsConfig.ServerName = host
			}
		}
		// If still no ServerName, use a generic one to satisfy TLS requirements
		if tlsConfig.ServerName == "" {
			tlsConfig.ServerName = "rdp-server"
		}
	}

	// When skipping validation, still try to verify if possible
	if insecureSkipVerify {
		// Set up fallback verification for basic connectivity
		tlsConfig.InsecureSkipVerify = true
		// Allow any cipher suite when skipping verification for maximum compatibility
		tlsConfig.CipherSuites = nil
	} else {
		// Enforce secure cipher suites when verification is enabled
		tlsConfig.CipherSuites = []uint16{
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		}
	}

	tlsConn := tls.Client(c.conn, tlsConfig)

	// Set handshake timeout to prevent hanging
	if tcpConn, ok := c.conn.(*net.TCPConn); ok {
		_ = tcpConn.SetReadDeadline(time.Now().Add(30 * time.Second))
		_ = tcpConn.SetWriteDeadline(time.Now().Add(30 * time.Second))
	}

	if err := tlsConn.Handshake(); err != nil {
		// Provide more specific error messages
		if insecureSkipVerify {
			return fmt.Errorf("TLS handshake failed even with validation skipped: %w", err)
		}

		// Suggest skipping validation for common certificate issues
		if strings.Contains(err.Error(), "certificate") || strings.Contains(err.Error(), "x509") {
			return fmt.Errorf("TLS certificate verification failed: %w. Consider using --skip-tls-validation for development environments", err)
		}

		// Handle the specific case where ServerName is missing
		if strings.Contains(err.Error(), "either ServerName or InsecureSkipVerify") {
			return fmt.Errorf("TLS configuration error: %w. When using IP addresses, either specify --tls-server-name or use --skip-tls-validation", err)
		}

		return fmt.Errorf("TLS handshake failed: %w", err)
	}

	// Clear any deadlines set for the handshake
	if tcpConn, ok := c.conn.(*net.TCPConn); ok {
		_ = tcpConn.SetReadDeadline(time.Time{})
		_ = tcpConn.SetWriteDeadline(time.Time{})
	}

	c.conn = tlsConn
	c.buffReader = bufio.NewReaderSize(c.conn, readBufferSize)

	return nil
}

func (c *Client) getServerName() string {
	if c.conn == nil {
		return ""
	}

	remoteAddr := c.conn.RemoteAddr().String()
	if remoteAddr == "" {
		return ""
	}

	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		// Fallback to extracting from hostname if available
		parts := strings.Split(remoteAddr, ":")
		if len(parts) > 0 {
			host = parts[0]
		} else {
			return ""
		}
	}

	// Clean up the hostname
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}

	// For IP addresses, we might not want to verify the hostname anyway
	// Return empty to let Go use the IP directly
	if net.ParseIP(host) != nil {
		return ""
	}

	// Validate hostname format
	if len(host) > 253 { // Maximum hostname length
		return ""
	}

	return host
}

func (c *Client) getMinTLSVersion(version string) uint16 {
	switch version {
	case "1.0":
		return tls.VersionTLS10
	case "1.1":
		return tls.VersionTLS11
	case "1.2":
		return tls.VersionTLS12
	case "1.3":
		return tls.VersionTLS13
	default:
		return tls.VersionTLS12 // Safe default
	}
}
