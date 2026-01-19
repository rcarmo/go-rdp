package rdp

import (
	"bufio"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/rcarmo/rdp-html5/internal/auth"
	"github.com/rcarmo/rdp-html5/internal/config"
)

// StartNLA performs Network Level Authentication using CredSSP/NTLMv2
func (c *Client) StartNLA() error {
	// First, establish TLS connection
	if err := c.startTLSForNLA(); err != nil {
		return fmt.Errorf("NLA TLS setup failed: %w", err)
	}

	// Parse domain from username if present (DOMAIN\user or user@domain)
	domain, user := c.parseDomainUser()
	log.Printf("NLA: Authenticating as domain=%q user=%q", domain, user)

	// Create NTLMv2 context
	ntlmCtx := auth.NewNTLMv2(domain, user, c.password)

	// Generate client nonce (32 bytes) - required for version 5+
	clientNonce := make([]byte, 32)
	if _, err := rand.Read(clientNonce); err != nil {
		return fmt.Errorf("NLA: failed to generate nonce: %w", err)
	}

	// Step 1: Send NTLM Negotiate message with client nonce
	negoMsg := ntlmCtx.GetNegotiateMessage()
	tsReq := auth.EncodeTSRequestWithNonce([][]byte{negoMsg}, nil, nil, clientNonce)

	if _, err := c.conn.Write(tsReq); err != nil {
		return fmt.Errorf("NLA: failed to send negotiate message: %w", err)
	}
	log.Printf("NLA: Sent negotiate message (%d bytes)", len(tsReq))

	// Step 2: Receive server challenge
	resp := make([]byte, 4096)
	n, err := c.conn.Read(resp)
	if err != nil {
		return fmt.Errorf("NLA: failed to read challenge: %w", err)
	}
	log.Printf("NLA: Received challenge (%d bytes)", n)

	tsResp, err := auth.DecodeTSRequest(resp[:n])
	if err != nil {
		return fmt.Errorf("NLA: failed to decode challenge: %w", err)
	}
	log.Printf("NLA: Server version=%d, errorCode=%d", tsResp.Version, tsResp.ErrorCode)

	if len(tsResp.NegoTokens) == 0 {
		return fmt.Errorf("NLA: no challenge token received from server")
	}

	// Step 3: Process challenge and get authenticate message
	authMsg, ntlmSec := ntlmCtx.GetAuthenticateMessage(tsResp.NegoTokens[0].Data)
	if authMsg == nil || ntlmSec == nil {
		return fmt.Errorf("NLA: failed to generate authenticate message")
	}

	// Get the server's public key from the TLS connection
	pubKey, err := c.getTLSPublicKey()
	if err != nil {
		return fmt.Errorf("NLA: failed to get TLS public key: %w", err)
	}
	log.Printf("NLA: Got TLS public key (%d bytes)", len(pubKey))

	// For version 5+, compute hash-based pubKeyAuth
	// SHA256(ClientServerHashMagic || clientNonce || publicKey)
	var pubKeyData []byte
	if tsResp.Version >= 5 {
		pubKeyData = auth.ComputeClientPubKeyAuth(tsResp.Version, pubKey, clientNonce)
		log.Printf("NLA: Using version %d hash-based pubKeyAuth (len=%d)", tsResp.Version, len(pubKeyData))
	} else {
		pubKeyData = pubKey
		log.Printf("NLA: Using version %d raw pubKey", tsResp.Version)
	}

	encryptedPubKey := ntlmSec.GssEncrypt(pubKeyData)
	log.Printf("NLA: Encrypted pubKeyAuth len=%d", len(encryptedPubKey))

	// Send authenticate message with encrypted public key and client nonce
	tsReq = auth.EncodeTSRequestWithNonce([][]byte{authMsg}, nil, encryptedPubKey, clientNonce)
	if _, err := c.conn.Write(tsReq); err != nil {
		return fmt.Errorf("NLA: failed to send authenticate message: %w", err)
	}
	log.Printf("NLA: Sent authenticate message (%d bytes), authMsg len=%d", len(tsReq), len(authMsg))

	// Step 4: Receive public key verification from server
	resp = make([]byte, 4096)
	n, err = c.conn.Read(resp)
	if err != nil {
		return fmt.Errorf("NLA: failed to read public key response: %w", err)
	}
	log.Printf("NLA: Received public key response (%d bytes)", n)

	tsResp, err = auth.DecodeTSRequest(resp[:n])
	if err != nil {
		return fmt.Errorf("NLA: failed to decode public key response: %w", err)
	}

	// Step 5: Send credentials
	domainBytes, userBytes, passBytes := ntlmCtx.GetEncodedCredentials()
	credentials := auth.EncodeCredentials(domainBytes, userBytes, passBytes)
	encryptedCreds := ntlmSec.GssEncrypt(credentials)

	tsReq = auth.EncodeTSRequest(nil, encryptedCreds, nil)
	if _, err := c.conn.Write(tsReq); err != nil {
		return fmt.Errorf("NLA: failed to send credentials: %w", err)
	}

	return nil
}

// startTLSForNLA establishes TLS connection for NLA
func (c *Client) startTLSForNLA() error {
	insecureSkipVerify := c.skipTLSValidation
	serverName := c.tlsServerName

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
	}

	if serverName == "" {
		serverName = c.getServerName()
	}

	// Windows RDP servers typically only support TLS 1.0-1.2, not TLS 1.3
	// Use TLS 1.2 max for better compatibility
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // RDP servers use self-signed certs
		MinVersion:         tls.VersionTLS10,
		MaxVersion:         tls.VersionTLS12, // Windows RDP doesn't support TLS 1.3
		ServerName:         serverName,
	}

	if tlsConfig.ServerName == "" {
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
		if strings.Contains(err.Error(), "certificate") || strings.Contains(err.Error(), "x509") {
			return fmt.Errorf("TLS certificate verification failed: %w", err)
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
// Per MS-CSSP, this must be the SubjectPublicKeyInfo from the certificate
func (c *Client) getTLSPublicKey() ([]byte, error) {
	tlsConn, ok := c.conn.(*tls.Conn)
	if !ok {
		return nil, fmt.Errorf("connection is not TLS")
	}

	state := tlsConn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return nil, fmt.Errorf("no peer certificates")
	}

	// Per MS-CSSP, we need the SubjectPublicKeyInfo (the raw DER-encoded public key)
	cert := state.PeerCertificates[0]
	return cert.RawSubjectPublicKeyInfo, nil
}

// parseDomainUser parses DOMAIN\user or user@domain format
func (c *Client) parseDomainUser() (domain, user string) {
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
