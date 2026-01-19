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
	log.Printf("NLA: Got TLS SubjectPublicKey (%d bytes, first 20: %x)", len(pubKey), pubKey[:min(20, len(pubKey))])

	// For version 5+, compute hash-based pubKeyAuth
	// SHA256(ClientServerHashMagic || clientNonce || publicKey)
	var pubKeyData []byte
	if tsResp.Version >= 5 {
		pubKeyData = auth.ComputeClientPubKeyAuth(tsResp.Version, pubKey, clientNonce)
		log.Printf("NLA: Using version %d hash-based pubKeyAuth (len=%d, first 20: %x)", tsResp.Version, len(pubKeyData), pubKeyData[:min(20, len(pubKeyData))])
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

	// Verify server's pubKeyAuth (for version 5+, this is a hash; for earlier versions, pubKey+1)
	if len(tsResp.PubKeyAuth) > 0 {
		decryptedPubKeyAuth := ntlmSec.GssDecrypt(tsResp.PubKeyAuth)
		if decryptedPubKeyAuth == nil {
			return fmt.Errorf("NLA: failed to decrypt server pubKeyAuth")
		}
		log.Printf("NLA: Decrypted server pubKeyAuth (%d bytes)", len(decryptedPubKeyAuth))

		// Verify the server's response
		if !auth.VerifyServerPubKeyAuth(tsResp.Version, decryptedPubKeyAuth, pubKey, clientNonce) {
			return fmt.Errorf("NLA: server pubKeyAuth verification failed")
		}
		log.Printf("NLA: Server pubKeyAuth verified successfully")
	}

	// Step 5: Send credentials
	domainBytes, userBytes, passBytes := ntlmCtx.GetEncodedCredentials()
	credentials := auth.EncodeCredentials(domainBytes, userBytes, passBytes)
	encryptedCreds := ntlmSec.GssEncrypt(credentials)
	log.Printf("NLA: Sending encrypted credentials (%d bytes)", len(encryptedCreds))

	tsReq = auth.EncodeTSRequest(nil, encryptedCreds, nil)
	if _, err := c.conn.Write(tsReq); err != nil {
		return fmt.Errorf("NLA: failed to send credentials: %w", err)
	}
	log.Printf("NLA: Credentials sent successfully")

	// Step 6: Wait for final server response (optional - some servers send a final TSRequest)
	// Set a short timeout for this read - if no data comes, credentials were accepted
	if tcpConn, ok := c.conn.(*tls.Conn); ok {
		tcpConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	}

	finalResp := make([]byte, 4096)
	n, err = c.conn.Read(finalResp)
	if err != nil {
		// Timeout is expected and OK - means server accepted credentials silently
		if ne, ok := err.(net.Error); ok && ne.Timeout() {
			log.Printf("NLA: No final response from server (timeout - expected)")
			// Clear the deadline
			if tcpConn, ok := c.conn.(*tls.Conn); ok {
				tcpConn.SetReadDeadline(time.Time{})
			}
			return nil
		}
		// Other errors might indicate authentication failure
		log.Printf("NLA: Error reading final response: %v", err)
		// Try to continue anyway - some servers don't send a final response
		if tcpConn, ok := c.conn.(*tls.Conn); ok {
			tcpConn.SetReadDeadline(time.Time{})
		}
		return nil
	}

	// Clear the deadline
	if tcpConn, ok := c.conn.(*tls.Conn); ok {
		tcpConn.SetReadDeadline(time.Time{})
	}

	// If we got data, check if it's an error response
	if n > 0 {
		log.Printf("NLA: Received final response (%d bytes)", n)
		finalTsResp, err := auth.DecodeTSRequest(finalResp[:n])
		if err == nil {
			if finalTsResp.ErrorCode != 0 {
				return fmt.Errorf("NLA: server returned error code: 0x%08X", finalTsResp.ErrorCode)
			}
			log.Printf("NLA: Final response indicates success (version=%d)", finalTsResp.Version)
		}
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
// Per MS-CSSP, this must be the SubjectPublicKey (NOT SubjectPublicKeyInfo)
// SubjectPublicKeyInfo = SEQUENCE { algorithm, subjectPublicKey }
// We need just the subjectPublicKey BIT STRING content
func (c *Client) getTLSPublicKey() ([]byte, error) {
	tlsConn, ok := c.conn.(*tls.Conn)
	if !ok {
		return nil, fmt.Errorf("connection is not TLS")
	}

	state := tlsConn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return nil, fmt.Errorf("no peer certificates")
	}

	cert := state.PeerCertificates[0]

	// Parse SubjectPublicKeyInfo to extract just SubjectPublicKey
	// SubjectPublicKeyInfo ::= SEQUENCE {
	//   algorithm AlgorithmIdentifier,
	//   subjectPublicKey BIT STRING
	// }
	// We need the raw bytes of the BIT STRING (including its unused bits prefix)
	spki := cert.RawSubjectPublicKeyInfo
	if len(spki) < 4 {
		return nil, fmt.Errorf("SubjectPublicKeyInfo too short")
	}

	// Parse outer SEQUENCE
	if spki[0] != 0x30 {
		return nil, fmt.Errorf("expected SEQUENCE tag for SubjectPublicKeyInfo")
	}

	// Parse length
	offset := 1
	seqLen, lenBytes := parseASN1Length(spki[offset:])
	offset += lenBytes
	if seqLen == 0 || offset+seqLen > len(spki) {
		return nil, fmt.Errorf("invalid SubjectPublicKeyInfo length")
	}

	// Skip AlgorithmIdentifier SEQUENCE
	if spki[offset] != 0x30 {
		return nil, fmt.Errorf("expected SEQUENCE tag for AlgorithmIdentifier")
	}
	algIdLen, algIdLenBytes := parseASN1Length(spki[offset+1:])
	offset += 1 + algIdLenBytes + algIdLen

	// Now at SubjectPublicKey BIT STRING
	if offset >= len(spki) || spki[offset] != 0x03 {
		return nil, fmt.Errorf("expected BIT STRING tag for SubjectPublicKey")
	}
	offset++ // skip tag

	bitStrLen, bitStrLenBytes := parseASN1Length(spki[offset:])
	offset += bitStrLenBytes

	if offset+bitStrLen > len(spki) {
		return nil, fmt.Errorf("SubjectPublicKey extends past end of SubjectPublicKeyInfo")
	}

	// Skip the "unused bits" byte (first byte of BIT STRING content)
	// FreeRDP's i2d_PublicKey returns just the raw key structure without this byte
	if bitStrLen < 1 {
		return nil, fmt.Errorf("SubjectPublicKey BIT STRING too short")
	}
	offset++ // skip unused bits byte
	bitStrLen--

	// Return just the raw public key DER (SEQUENCE { modulus, exponent } for RSA)
	return spki[offset : offset+bitStrLen], nil
}

// parseASN1Length parses ASN.1 DER length encoding
// Returns the length value and the number of bytes consumed
func parseASN1Length(data []byte) (int, int) {
	if len(data) == 0 {
		return 0, 0
	}
	if data[0] < 128 {
		return int(data[0]), 1
	}
	numBytes := int(data[0] & 0x7F)
	if numBytes == 0 || numBytes > 4 || numBytes >= len(data) {
		return 0, 1
	}
	length := 0
	for i := 0; i < numBytes; i++ {
		length = (length << 8) | int(data[1+i])
	}
	return length, 1 + numBytes
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
