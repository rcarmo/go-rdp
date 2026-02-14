// Package handler implements HTTP handlers for the RDP HTML5 gateway,
// including WebSocket connection management and RDP session proxying.
package handler

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/websocket"

	"github.com/rcarmo/go-rdp/internal/config"
	"github.com/rcarmo/go-rdp/internal/logging"
	"github.com/rcarmo/go-rdp/internal/protocol/audio"
	"github.com/rcarmo/go-rdp/internal/protocol/pdu"
	"github.com/rcarmo/go-rdp/internal/rdp"
)

type rdpConn interface {
	GetUpdate() (*rdp.Update, error)
	SendInputEvent(data []byte) error
}

// capabilitiesGetter interface for testing
type capabilitiesGetter interface {
	GetServerCapabilities() *rdp.ServerCapabilityInfo
	IsDisplayControlReady() bool
}

// connectionRequest represents credentials sent via WebSocket
type connectionRequest struct {
	Type     string `json:"type"`
	Host     string `json:"host"`
	User     string `json:"user"`
	Password string `json:"password"`
}

// Connect handles WebSocket connections for RDP sessions.
func Connect(w http.ResponseWriter, r *http.Request) {
	// Check origin
	origin := r.Header.Get("Origin")
	if origin != "" && !isAllowedOrigin(origin) {
		http.Error(w, "Origin not allowed", http.StatusForbidden)
		return
	}

	// Create websocket handler
	handler := func(wsConn *websocket.Conn) {
		handleWebSocket(wsConn, r)
	}

	// Configure and serve websocket
	server := websocket.Server{
		Handler: handler,
		Handshake: func(config *websocket.Config, r *http.Request) error {
			// Accept any origin that passed our check
			config.Origin, _ = websocket.Origin(config, r)
			return nil
		},
	}
	server.ServeHTTP(w, r)
}

// connectionParams holds validated connection parameters
type connectionParams struct {
	width      int
	height     int
	colorDepth int
	disableNLA bool
	enableAudio bool
}

// parseConnectionParams extracts and validates connection parameters from the request.
func parseConnectionParams(r *http.Request) (*connectionParams, error) {
	width, err := strconv.Atoi(r.URL.Query().Get("width"))
	if err != nil || width <= 0 || width > 8192 {
		return nil, errors.New("invalid width parameter (must be 1-8192)")
	}

	height, err := strconv.Atoi(r.URL.Query().Get("height"))
	if err != nil || height <= 0 || height > 8192 {
		return nil, errors.New("invalid height parameter (must be 1-8192)")
	}

	colorDepth := 16 // default to 16-bit
	if cdStr := r.URL.Query().Get("colorDepth"); cdStr != "" {
		if cd, err := strconv.Atoi(cdStr); err == nil && (cd == 8 || cd == 15 || cd == 16 || cd == 24 || cd == 32) {
			colorDepth = cd
		}
	}

	return &connectionParams{
		width:       width,
		height:      height,
		colorDepth:  colorDepth,
		disableNLA:  r.URL.Query().Get("disableNLA") == "true",
		enableAudio: r.URL.Query().Get("audio") == "true",
	}, nil
}

// receiveCredentials waits for and validates credentials sent via WebSocket.
func receiveCredentials(wsConn *websocket.Conn) (*connectionRequest, error) {
	// Set read deadline for credentials
	if err := wsConn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return nil, errors.New("failed to set read deadline")
	}

	var credMsg []byte
	if err := websocket.Message.Receive(wsConn, &credMsg); err != nil {
		return nil, errors.New("failed to receive credentials")
	}

	// Limit credential message size to prevent DoS (1MB max)
	if len(credMsg) > 1024*1024 {
		return nil, errors.New("credentials message too large")
	}

	// Clear read deadline
	if err := wsConn.SetReadDeadline(time.Time{}); err != nil {
		return nil, errors.New("failed to clear read deadline")
	}

	var credentials connectionRequest
	if err := json.Unmarshal(credMsg, &credentials); err != nil {
		return nil, errors.New("invalid credentials format")
	}

	if credentials.Type != "credentials" {
		return nil, errors.New("expected credentials message")
	}

	// Validate hostname length (max 253 per DNS spec)
	if len(credentials.Host) == 0 || len(credentials.Host) > 253 {
		return nil, errors.New("invalid hostname")
	}

	// Validate username length (Windows max is 256)
	if len(credentials.User) == 0 || len(credentials.User) > 256 {
		return nil, errors.New("invalid username")
	}

	// Validate password length (reasonable max, prevent memory exhaustion)
	if len(credentials.Password) > 1024 {
		return nil, errors.New("password too long")
	}

	return &credentials, nil
}

// setupRDPClient creates and configures an RDP client with the given parameters.
func setupRDPClient(creds *connectionRequest, params *connectionParams) (*rdp.Client, error) {
	rdpClient, err := rdp.NewClient(creds.Host, creds.User, creds.Password, params.width, params.height, params.colorDepth)
	if err != nil {
		return nil, err
	}

	// Set TLS configuration from server config
	cfg := config.GetGlobalConfig()
	if cfg == nil {
		cfg, err = config.Load()
		if err != nil {
			logging.Debug("Failed to load config for TLS settings: %v", err)
			cfg = &config.Config{}
		}
	}

	rdpClient.SetTLSConfig(cfg.Security.SkipTLSValidation, cfg.Security.TLSServerName)

	// Use NLA unless explicitly disabled by client or server config
	useNLA := cfg.Security.UseNLA && !params.disableNLA
	rdpClient.SetUseNLA(useNLA)
	if params.disableNLA {
		logging.Info("NLA disabled for this connection")
	}

	// Enable audio if requested
	if params.enableAudio {
		rdpClient.EnableAudio()
		logging.Info("Audio redirection enabled")
	}

	// Enable display control for dynamic resize
	rdpClient.EnableDisplayControl()
	logging.Debug("Display control enabled")

	// Enable UDP transport if configured (experimental)
	if cfg.RDP.EnableUDP {
		rdpClient.EnableMultitransport(true)
		logging.Info("UDP transport enabled (experimental)")
	}

	// Enable RemoteFX-Image codec if configured
	if cfg.RDP.EnableRFX {
		rdpClient.SetEnableRFX(true)
	}

	return rdpClient, nil
}

// startBidirectionalRelay manages the goroutines that relay data between WebSocket and RDP.
func startBidirectionalRelay(ctx context.Context, cancel context.CancelFunc, wsConn *websocket.Conn, rdpClient *rdp.Client, wsMu *sync.Mutex, enableAudio bool) {
	// Set up audio callback to forward audio data to browser
	if enableAudio && rdpClient.GetAudioHandler() != nil {
		rdpClient.GetAudioHandler().SetCallback(func(data []byte, format *audio.AudioFormat, timestamp uint16) {
			sendAudioDataWithMutex(wsConn, wsMu, data, format, timestamp)
		})
	}

	// Send server capabilities info to browser
	sendCapabilitiesInfoWithMutex(wsConn, wsMu, rdpClient)

	// Use WaitGroup to ensure clean goroutine shutdown
	var cancelOnce sync.Once
	safeCancel := func() { cancelOnce.Do(cancel) }
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		wsToRdp(ctx, wsConn, rdpClient, safeCancel)
	}()
	rdpToWsWithMutex(ctx, rdpClient, wsConn, wsMu)

	// Cancel context to signal wsToRdp to exit
	safeCancel()

	// Wait for wsToRdp goroutine to finish (with timeout)
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		logging.Warn("Timeout waiting for wsToRdp goroutine to exit")
	}
}

func handleWebSocket(wsConn *websocket.Conn, r *http.Request) {
	defer func() { _ = wsConn.Close() }()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Parse and validate connection parameters
	params, err := parseConnectionParams(r)
	if err != nil {
		logging.Error("Invalid params: %v", err)
		sendError(wsConn, err.Error())
		return
	}

	// Receive and validate credentials
	credentials, err := receiveCredentials(wsConn)
	if err != nil {
		logging.Error("Credentials error: %v", err)
		sendError(wsConn, err.Error())
		return
	}

	// Create and configure RDP client
	rdpClient, err := setupRDPClient(credentials, params)
	if err != nil {
		logging.Error("RDP init: %v", err)
		sendError(wsConn, "Connection failed")
		return
	}
	defer func() { _ = rdpClient.Close() }()

	// Connect to RDP server
	if err = rdpClient.Connect(); err != nil {
		logging.Error("RDP connect: %v", err)
		sendError(wsConn, "Connection failed")
		return
	}

	// Per-connection mutex for WebSocket writes
	var wsMu sync.Mutex

	// Start bidirectional data relay
	startBidirectionalRelay(ctx, cancel, wsConn, rdpClient, &wsMu, params.enableAudio)
}

// resizeRequest represents a display resize request from the browser
type resizeRequest struct {
	Type   string `json:"type"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// resizer interface for display control
type resizer interface {
	RequestResize(width, height int) error
	IsDisplayControlReady() bool
}

func wsToRdp(ctx context.Context, wsConn *websocket.Conn, rdpConn rdpConn, cancel context.CancelFunc) {
	defer func() {
		if r := recover(); r != nil {
			logging.Error("Panic in wsToRdp: %v", r)
		}
	}()

	// Check if rdpConn supports resize
	resizerConn, supportsResize := rdpConn.(resizer)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Apply a read deadline to avoid hung connections keeping goroutines alive
		_ = wsConn.SetReadDeadline(time.Now().Add(30 * time.Second))

		var data []byte
		if err := websocket.Message.Receive(wsConn, &data); err != nil {
			if err == io.EOF || strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			logging.Error("Error reading message from WS: %v", err)
			cancel()
			return
		}

		// Check if this is a JSON message (starts with '{')
		if len(data) > 0 && data[0] == '{' {
			var msg map[string]interface{}
			if err := json.Unmarshal(data, &msg); err == nil {
				if msgType, ok := msg["type"].(string); ok && msgType == "resize" {
					if supportsResize {
						var req resizeRequest
						if err := json.Unmarshal(data, &req); err == nil {
							if resizerConn.IsDisplayControlReady() {
								if err := resizerConn.RequestResize(req.Width, req.Height); err != nil {
									logging.Debug("Resize request failed: %v", err)
								} else {
									logging.Info("Resize requested: %dx%d", req.Width, req.Height)
								}
							} else {
								logging.Debug("Display control not ready, resize ignored")
							}
						}
					}
					continue // Don't send resize as input event
				}
			}
		}

		if err := rdpConn.SendInputEvent(data); err != nil {
			logging.Error("Failed writing to RDP: %v", err)
			return
		}
	}
}

func rdpToWsWithMutex(ctx context.Context, rdpConn rdpConn, wsConn *websocket.Conn, wsMu *sync.Mutex) {
	defer func() {
		if r := recover(); r != nil {
			logging.Error("Panic in rdpToWs: %v", r)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		update, err := rdpConn.GetUpdate()
		switch {
		case err == nil:
		case errors.Is(err, pdu.ErrDeactivateAll):
			return
		default:
			logging.Error("Get update: %v", err)
			return
		}

		wsMu.Lock()
		err = websocket.Message.Send(wsConn, update.Data)
		wsMu.Unlock()

		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			logging.Error("Failed sending message to WS: %v", err)
			return
		}
	}
}

// sendCapabilitiesInfoWithMutex sends server capabilities to the browser
func sendCapabilitiesInfoWithMutex(wsConn *websocket.Conn, wsMu *sync.Mutex, rdpClient capabilitiesGetter) {
	caps := rdpClient.GetServerCapabilities()
	if caps == nil {
		return
	}

	displayControlReady := rdpClient.IsDisplayControlReady()

	logging.Info("Session: NLA=%v audio=%v channels=%v colorDepth=%d desktop=%s codecs=%v displayControl=%v",
		caps.UseNLA, caps.AudioEnabled, caps.Channels, caps.ColorDepth, caps.DesktopSize, caps.BitmapCodecs, displayControlReady)

	msg := buildCapabilitiesMessage(caps, displayControlReady)

	wsMu.Lock()
	defer wsMu.Unlock()
	if err := websocket.Message.Send(wsConn, msg); err != nil {
		logging.Error("Failed to send capabilities info: %v", err)
	}
}

// buildCapabilitiesMessage creates the capabilities JSON message
func buildCapabilitiesMessage(caps *rdp.ServerCapabilityInfo, displayControlReady bool) []byte {
	logLevel := strings.ToLower(logging.GetLevelString())
	
	payload := map[string]any{
		"type":                "capabilities",
		"codecs":              caps.BitmapCodecs,
		"surfaceCommands":     caps.SurfaceCommands,
		"colorDepth":          caps.ColorDepth,
		"desktopSize":         caps.DesktopSize,
		"multifragmentSize":   caps.MultifragmentSize,
		"largePointer":        caps.LargePointer,
		"frameAcknowledge":    caps.FrameAcknowledge,
		"useNLA":              caps.UseNLA,
		"audioEnabled":        caps.AudioEnabled,
		"channels":            caps.Channels,
		"logLevel":            logLevel,
		"displayControlReady": displayControlReady,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		logging.Error("Failed to marshal capabilities: %v", err)
		return nil
	}

	msg := make([]byte, 1+len(jsonData))
	msg[0] = 0xFF
	copy(msg[1:], jsonData)
	return msg
}

func codecListToJSON(codecs []string) string {
	if len(codecs) == 0 {
		return ""
	}
	quoted := make([]string, len(codecs))
	for i, c := range codecs {
		quoted[i] = `"` + c + `"`
	}
	return strings.Join(quoted, ",")
}

// errorMessage is the JSON structure for WebSocket error responses
type errorMessage struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// sendError sends an error message to the client via WebSocket
func sendError(wsConn *websocket.Conn, message string) {
	errMsg, err := json.Marshal(errorMessage{Type: "error", Message: message})
	if err != nil {
		logging.Error("Failed to marshal error message: %v", err)
		return
	}
	if err := websocket.Message.Send(wsConn, string(errMsg)); err != nil {
		logging.Error("Failed to send error message: %v", err)
	}
}

func isAllowedOrigin(origin string) bool {
	if origin == "" {
		return false
	}

	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return false
	}
	return true
}

func IsOriginAllowed(origin string, allowedOrigins []string, host string) bool {
	if origin == "" {
		return false
	}

	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return false
	}
	_ = allowedOrigins
	_ = host
	return true
}

// Audio message types for WebSocket
const (
	AudioMsgTypeData   = 0x01 // Audio PCM data
	AudioMsgTypeFormat = 0x02 // Audio format info
)

// sendAudioDataWithMutex sends audio data to the browser over WebSocket with per-connection mutex
// Format: [0xFE][msgType][timestamp 2 bytes][format info if type=format][data]
func sendAudioDataWithMutex(wsConn *websocket.Conn, wsMu *sync.Mutex, data []byte, format *audio.AudioFormat, timestamp uint16) {
	if len(data) == 0 {
		return
	}

	// Build audio message
	// Header: 0xFE (audio marker), msgType, timestamp (2 bytes LE)
	headerSize := 4
	
	// For format messages, include format info
	var formatInfo []byte
	if format != nil {
		// Format: channels (2), sampleRate (4), bitsPerSample (2), formatTag (2)
		formatInfo = make([]byte, 10)
		binary.LittleEndian.PutUint16(formatInfo[0:2], format.Channels)
		binary.LittleEndian.PutUint32(formatInfo[2:6], format.SamplesPerSec)
		binary.LittleEndian.PutUint16(formatInfo[6:8], format.BitsPerSample)
		binary.LittleEndian.PutUint16(formatInfo[8:10], format.FormatTag)
	}

	msg := make([]byte, headerSize+len(formatInfo)+len(data))
	msg[0] = 0xFE // Audio marker
	msg[1] = AudioMsgTypeData
	binary.LittleEndian.PutUint16(msg[2:4], timestamp)
	
	offset := headerSize
	if len(formatInfo) > 0 {
		msg[1] = AudioMsgTypeFormat // Include format
		copy(msg[offset:], formatInfo)
		offset += len(formatInfo)
	}
	copy(msg[offset:], data)

	wsMu.Lock()
	err := websocket.Message.Send(wsConn, msg)
	wsMu.Unlock()

	if err != nil {
		logging.Debug("Failed to send audio data: %v", err)
	}
}
