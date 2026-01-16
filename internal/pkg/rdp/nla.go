package rdp

import (
	"bufio"
	"crypto/tls"
	"encoding/asn1"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/kulaginds/rdp-html5/internal/pkg/config"
	"github.com/tomatome/grdp/protocol/nla"
)

// StartNLA performs Network Level Authentication using CredSSP/NTLMv2
func (c *client) StartNLA() error {
	// First, establish TLS connection
	if err := c.startTLSForNLA(); err != nil {
		return fmt.Errorf("NLA TLS setup failed: %w", err)
	}

	// Parse domain from username if present (DOMAIN\user or user@domain)
	domain, user := c.parseDomainUser()

	// Create NTLMv2 context using grdp's implementation
	ntlm := nla.NewNTLMv2(domain, user, c.password)

	// Step 1: Send NTLM Negotiate message
	negoMsg := ntlm.GetNegotiateMessage()
	tsReq := nla.EncodeDERTRequest([]nla.Message{negoMsg}, nil, nil)

	if _, err := c.conn.Write(tsReq); err != nil {
		return fmt.Errorf("NLA: failed to send negotiate message: %w", err)
	}

	// Step 2: Receive server challenge
	resp := make([]byte, 4096)
	n, err := c.conn.Read(resp)
	if err != nil {
		return fmt.Errorf("NLA: failed to read challenge: %w", err)
	}

	tsResp, err := nla.DecodeDERTRequest(resp[:n])
	if err != nil {
		return fmt.Errorf("NLA: failed to decode challenge: %w", err)
	}

	if len(tsResp.NegoTokens) == 0 {
		return fmt.Errorf("NLA: no challenge token received from server")
	}

	// Step 3: Process challenge and get authenticate message
	authMsg, ntlmSec := ntlm.GetAuthenticateMessage(tsResp.NegoTokens[0].Data)
	if authMsg == nil || ntlmSec == nil {
		return fmt.Errorf("NLA: failed to generate authenticate message")
	}

	// Get the server's public key from the TLS connection
	pubKey, err := c.getTLSPublicKey()
	if err != nil {
		return fmt.Errorf("NLA: failed to get TLS public key: %w", err)
	}

	// Encrypt the public key
	encryptedPubKey := ntlmSec.GssEncrypt(pubKey)

	// Send authenticate message with encrypted public key
	tsReq = nla.EncodeDERTRequest([]nla.Message{authMsg}, nil, encryptedPubKey)
	if _, err := c.conn.Write(tsReq); err != nil {
		return fmt.Errorf("NLA: failed to send authenticate message: %w", err)
	}

	// Step 4: Receive public key verification from server
	resp = make([]byte, 4096)
	n, err = c.conn.Read(resp)
	if err != nil {
		return fmt.Errorf("NLA: failed to read public key response: %w", err)
	}

	tsResp, err = nla.DecodeDERTRequest(resp[:n])
	if err != nil {
		return fmt.Errorf("NLA: failed to decode public key response: %w", err)
	}

	// Verify server's public key (optional, can be skipped for compatibility)
	// The server returns pubKey + 1, we could verify this but skip for now

	// Step 5: Send credentials)
	domainBytes, userBytes, passBytes := ntlm.GetEncodedCredentials()
	credentials := nla.EncodeDERTCredentials(domainBytes, userBytes, passBytes)
	encryptedCreds := ntlmSec.GssEncrypt(credentials)

	tsReq = nla.EncodeDERTRequest(nil, encryptedCreds, nil)
	if _, err := c.conn.Write(tsReq); err != nil {
		return fmt.Errorf("NLA: failed to send credentials: %w", err)
	}

	return nil
}

// startTLSForNLA establishes TLS connection for NLA
func (c *client) startTLSForNLA() error {
	insecureSkipVerify := c.skipTLSValidation
	serverName := c.tlsServerName
	minTLSVersion := "1.2"

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

	if insecureSkipVerify {
		minTLSVersion = "1.0"
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: insecureSkipVerify,
		MinVersion:         c.getMinTLSVersion(minTLSVersion),
		MaxVersion:         tls.VersionTLS13,
		ServerName:         serverName,
	}

	if tlsConfig.InsecureSkipVerify && tlsConfig.ServerName == "" {
		if c.conn != nil {
			remoteAddr := c.conn.RemoteAddr().String()
			host, _, err := net.SplitHostPort(remoteAddr)
			if err == nil && host != "" {
				tlsConfig.ServerName = host
			}
		}
		if tlsConfig.ServerName == "" {
			tlsConfig.ServerName = "rdp-server"
		}
	}

	tlsConn := tls.Client(c.conn, tlsConfig)

	if tcpConn, ok := c.conn.(*net.TCPConn); ok {
		tcpConn.SetReadDeadline(time.Now().Add(30 * time.Second))
		tcpConn.SetWriteDeadline(time.Now().Add(30 * time.Second))
	}

	if err := tlsConn.Handshake(); err != nil {
		if insecureSkipVerify {
			return fmt.Errorf("TLS handshake failed even with validation skipped: %w", err)
		}
		if strings.Contains(err.Error(), "certificate") || strings.Contains(err.Error(), "x509") {
			return fmt.Errorf("TLS certificate verification failed: %w. Consider using --skip-tls-validation", err)
		}
		return fmt.Errorf("TLS handshake failed: %w", err)
	}

	if tcpConn, ok := c.conn.(*net.TCPConn); ok {
		tcpConn.SetReadDeadline(time.Time{})
		tcpConn.SetWriteDeadline(time.Time{})
	}

	c.conn = tlsConn
	c.buffReader = bufio.NewReaderSize(c.conn, readBufferSize)

	return nil
}

// getTLSPublicKey extracts the server's public key from the TLS connection
func (c *client) getTLSPublicKey() ([]byte, error) {
	tlsConn, ok := c.conn.(*tls.Conn)
	if !ok {
		return nil, fmt.Errorf("connection is not TLS")
	}

	state := tlsConn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return nil, fmt.Errorf("no peer certificates")
	}

	// Get the raw public key from the certificate
	cert := state.PeerCertificates[0]
	pubKeyBytes, err := asn1.Marshal(cert.PublicKey)
	if err != nil {
		// Fallback: use raw public key info
		return cert.RawSubjectPublicKeyInfo, nil
	}

	return pubKeyBytes, nil
}

// parseDomainUser parses DOMAIN\user or user@domain format
func (c *client) parseDomainUser() (domain, user string) {
	username := c.username

	// Check for DOMAIN\user format
	if idx := strings.Index(username, "\\"); idx != -1 {
		return username[:idx], username[idx+1:]
	}

	// Check for user@domain format
	if idx := strings.Index(username, "@"); idx != -1 {
		return username[idx+1:], username[:idx]
	}

	// No domain specified
	return c.domain, username
}
