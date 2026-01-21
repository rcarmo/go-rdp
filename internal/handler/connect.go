package handler

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/websocket"

	"github.com/rcarmo/rdp-html5/internal/config"
	"github.com/rcarmo/rdp-html5/internal/logging"
	"github.com/rcarmo/rdp-html5/internal/protocol/audio"
	"github.com/rcarmo/rdp-html5/internal/protocol/pdu"
	"github.com/rcarmo/rdp-html5/internal/rdp"
)

type rdpConn interface {
	GetUpdate() (*rdp.Update, error)
	SendInputEvent(data []byte) error
}

// capabilitiesGetter interface for testing
type capabilitiesGetter interface {
	GetServerCapabilities() *rdp.ServerCapabilityInfo
}

// connectionRequest represents credentials sent via WebSocket
type connectionRequest struct {
	Type     string `json:"type"`
	Host     string `json:"host"`
	User     string `json:"user"`
	Password string `json:"password"`
}

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

func handleWebSocket(wsConn *websocket.Conn, r *http.Request) {
	defer func() { _ = wsConn.Close() }()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	width, err := strconv.Atoi(r.URL.Query().Get("width"))
	if err != nil || width <= 0 || width > 8192 {
		logging.Error("Invalid width: %v (must be 1-8192)", r.URL.Query().Get("width"))
		sendError(wsConn, "Invalid width parameter")
		return
	}

	height, err := strconv.Atoi(r.URL.Query().Get("height"))
	if err != nil || height <= 0 || height > 8192 {
		logging.Error("Invalid height: %v (must be 1-8192)", r.URL.Query().Get("height"))
		sendError(wsConn, "Invalid height parameter")
		return
	}

	colorDepth := 16 // default to 16-bit
	if cdStr := r.URL.Query().Get("colorDepth"); cdStr != "" {
		if cd, err := strconv.Atoi(cdStr); err == nil && (cd == 8 || cd == 15 || cd == 16 || cd == 24 || cd == 32) {
			colorDepth = cd
		}
	}

	// Check if NLA should be disabled for this connection
	disableNLA := r.URL.Query().Get("disableNLA") == "true"

	// Check if audio should be enabled
	enableAudio := r.URL.Query().Get("audio") == "true"

	// Wait for credentials via WebSocket message (more secure than URL params)
	var credentials connectionRequest
	
	// Set read deadline for credentials
	if err := wsConn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		logging.Error("Set read deadline: %v", err)
		return
	}
	
	var credMsg []byte
	if err := websocket.Message.Receive(wsConn, &credMsg); err != nil {
		logging.Error("Receive credentials: %v", err)
		sendError(wsConn, "Failed to receive credentials")
		return
	}
	
	// Limit credential message size to prevent DoS (1MB max)
	if len(credMsg) > 1024*1024 {
		logging.Error("Credentials message too large: %d bytes", len(credMsg))
		sendError(wsConn, "Credentials message too large")
		return
	}
	
	// Clear read deadline
	if err := wsConn.SetReadDeadline(time.Time{}); err != nil {
		logging.Error("Clear read deadline: %v", err)
		return
	}
	
	if err := json.Unmarshal(credMsg, &credentials); err != nil {
		logging.Error("Parse credentials: %v", err)
		sendError(wsConn, "Invalid credentials format")
		return
	}
	
	if credentials.Type != "credentials" {
		logging.Error("Invalid message type: %s", credentials.Type)
		sendError(wsConn, "Expected credentials message")
		return
	}
	
	host := credentials.Host
	user := credentials.User
	password := credentials.Password
	
	// Validate hostname length (max 253 per DNS spec)
	if len(host) == 0 || len(host) > 253 {
		sendError(wsConn, "Invalid hostname")
		return
	}
	
	// Validate username length (Windows max is 256)
	if len(user) == 0 || len(user) > 256 {
		sendError(wsConn, "Invalid username")
		return
	}
	
	// Validate password length (reasonable max, prevent memory exhaustion)
	if len(password) > 1024 {
		sendError(wsConn, "Password too long")
		return
	}

	rdpClient, err := rdp.NewClient(host, user, password, width, height, colorDepth)
	if err != nil {
		logging.Error("RDP init: %v", err)
		// Generic error message to avoid information leakage
		sendError(wsConn, "Connection failed")
		return
	}
	defer func() { _ = rdpClient.Close() }()

	// Set TLS configuration from server config
	cfg := config.GetGlobalConfig()
	if cfg == nil {
		var err error
		cfg, err = config.Load()
		if err != nil {
			logging.Debug("Failed to load config for TLS settings: %v", err)
			cfg = &config.Config{}
		}
	}

	rdpClient.SetTLSConfig(cfg.Security.SkipTLSValidation, cfg.Security.TLSServerName)
	
	// Use NLA unless explicitly disabled by client or server config
	useNLA := cfg.Security.UseNLA && !disableNLA
	rdpClient.SetUseNLA(useNLA)
	if disableNLA {
		logging.Info("NLA disabled for this connection")
	}
	
	// Enable audio if requested
	if enableAudio {
		rdpClient.EnableAudio()
		logging.Info("Audio redirection enabled")
	}

	if err = rdpClient.Connect(); err != nil {
		logging.Error("RDP connect: %v", err)
		return
	}
	
	// Set up audio callback to forward audio data to browser
	if enableAudio && rdpClient.GetAudioHandler() != nil {
		rdpClient.GetAudioHandler().SetCallback(func(data []byte, format *audio.AudioFormat, timestamp uint16) {
			sendAudioData(wsConn, data, format, timestamp)
		})
	}

	// Send server capabilities info to browser
	sendCapabilitiesInfo(wsConn, rdpClient)

	// Use WaitGroup to ensure clean goroutine shutdown
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		wsToRdp(ctx, wsConn, rdpClient, cancel)
	}()
	rdpToWs(ctx, rdpClient, wsConn)
	
	// Wait for wsToRdp goroutine to finish
	wg.Wait()
}

func wsToRdp(ctx context.Context, wsConn *websocket.Conn, rdpConn rdpConn, cancel context.CancelFunc) {
	defer cancel()
	defer func() {
		if r := recover(); r != nil {
			logging.Error("Panic in wsToRdp: %v", r)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		var data []byte
		if err := websocket.Message.Receive(wsConn, &data); err != nil {
			if err == io.EOF || strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			logging.Error("Error reading message from WS: %v", err)
			return
		}

		if err := rdpConn.SendInputEvent(data); err != nil {
			logging.Error("Failed writing to RDP: %v", err)
			return
		}
	}
}

var wsMutex sync.Mutex

func rdpToWs(ctx context.Context, rdpConn rdpConn, wsConn *websocket.Conn) {
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
		case errors.Is(err, pdu.ErrDeactiateAll):
			return
		default:
			logging.Error("Get update: %v", err)
			return
		}

		wsMutex.Lock()
		err = websocket.Message.Send(wsConn, update.Data)
		wsMutex.Unlock()

		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			logging.Error("Failed sending message to WS: %v", err)
			return
		}
	}
}

// sendCapabilitiesInfo sends server capabilities to the browser
func sendCapabilitiesInfo(wsConn *websocket.Conn, rdpClient capabilitiesGetter) {
	caps := rdpClient.GetServerCapabilities()
	if caps == nil {
		return
	}

	logging.Info("Server Capabilities: codecs=%v surfaceCmds=%v colorDepth=%d desktop=%s multifrag=%d largePtr=%v frameAck=%v",
		caps.BitmapCodecs, caps.SurfaceCommands, caps.ColorDepth, caps.DesktopSize,
		caps.MultifragmentSize, caps.LargePointer, caps.FrameAcknowledge)

	msg := buildCapabilitiesMessage(caps)

	wsMutex.Lock()
	defer wsMutex.Unlock()
	if err := websocket.Message.Send(wsConn, msg); err != nil {
		logging.Error("Failed to send capabilities info: %v", err)
	}
}

// buildCapabilitiesMessage creates the capabilities JSON message
func buildCapabilitiesMessage(caps *rdp.ServerCapabilityInfo) []byte {
	logLevel := strings.ToLower(logging.GetLevelString())
	
	jsonData := fmt.Sprintf(`{"type":"capabilities","codecs":[%s],"surfaceCommands":%t,"colorDepth":%d,"desktopSize":"%s","multifragmentSize":%d,"largePointer":%t,"frameAcknowledge":%t,"logLevel":"%s"}`,
		codecListToJSON(caps.BitmapCodecs),
		caps.SurfaceCommands,
		caps.ColorDepth,
		caps.DesktopSize,
		caps.MultifragmentSize,
		caps.LargePointer,
		caps.FrameAcknowledge,
		logLevel)

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

// sendError sends an error message to the client via WebSocket
func sendError(wsConn *websocket.Conn, message string) {
	errMsg := fmt.Sprintf(`{"type":"error","message":"%s"}`, message)
	if err := websocket.Message.Send(wsConn, errMsg); err != nil {
		logging.Error("Failed to send error message: %v", err)
	}
}

func isAllowedOrigin(origin string) bool {
	if origin == "" {
		return false
	}

	normalized := strings.TrimPrefix(strings.TrimPrefix(origin, "http://"), "https://")
	normalized = strings.TrimSuffix(normalized, "/")

	allowed := os.Getenv("ALLOWED_ORIGINS")
	if allowed == "" {
		return true
	}

	if strings.HasPrefix(normalized, "localhost") || strings.HasPrefix(normalized, "127.0.0.1") {
		return true
	}

	for _, entry := range strings.Split(allowed, ",") {
		candidate := strings.TrimSpace(entry)
		if candidate == "" {
			continue
		}
		if candidate == origin || candidate == normalized {
			return true
		}
		if strings.TrimPrefix(candidate, "http://") == normalized || strings.TrimPrefix(candidate, "https://") == normalized {
			return true
		}
	}

	return false
}

// Audio message types for WebSocket
const (
	AudioMsgTypeData   = 0x01 // Audio PCM data
	AudioMsgTypeFormat = 0x02 // Audio format info
)

// sendAudioData sends audio data to the browser over WebSocket
// Format: [0xFE][msgType][timestamp 2 bytes][format info if type=format][data]
func sendAudioData(wsConn *websocket.Conn, data []byte, format *audio.AudioFormat, timestamp uint16) {
	if len(data) == 0 {
		return
	}

	// Build audio message
	// Header: 0xFE (audio marker), msgType, timestamp (2 bytes LE)
	headerSize := 4
	
	// For format messages, include format info
	var formatInfo []byte
	if format != nil {
		// Format: channels (2), sampleRate (4), bitsPerSample (2)
		formatInfo = make([]byte, 8)
		binary.LittleEndian.PutUint16(formatInfo[0:2], format.Channels)
		binary.LittleEndian.PutUint32(formatInfo[2:6], format.SamplesPerSec)
		binary.LittleEndian.PutUint16(formatInfo[6:8], format.BitsPerSample)
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

	wsMutex.Lock()
	err := websocket.Message.Send(wsConn, msg)
	wsMutex.Unlock()

	if err != nil {
		logging.Debug("Failed to send audio data: %v", err)
	}
}
