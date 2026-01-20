package handler

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/net/websocket"

	"github.com/rcarmo/rdp-html5/internal/config"
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
	defer wsConn.Close()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	width, err := strconv.Atoi(r.URL.Query().Get("width"))
	if err != nil {
		log.Println(fmt.Errorf("get width: %w", err))
		return
	}

	height, err := strconv.Atoi(r.URL.Query().Get("height"))
	if err != nil {
		log.Println(fmt.Errorf("get height: %w", err))
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

	host := r.URL.Query().Get("host")
	user := r.URL.Query().Get("user")
	password := r.URL.Query().Get("password")

	rdpClient, err := rdp.NewClient(host, user, password, width, height, colorDepth)
	if err != nil {
		log.Println(fmt.Errorf("rdp init: %w", err))
		return
	}
	defer rdpClient.Close()

	// Set TLS configuration from server config
	cfg := config.GetGlobalConfig()
	if cfg == nil {
		var err error
		cfg, err = config.Load()
		if err != nil {
			log.Printf("Failed to load config for TLS settings: %v", err)
			cfg = &config.Config{}
		}
	}

	rdpClient.SetTLSConfig(cfg.Security.SkipTLSValidation, cfg.Security.TLSServerName)
	
	// Use NLA unless explicitly disabled by client or server config
	useNLA := cfg.Security.UseNLA && !disableNLA
	rdpClient.SetUseNLA(useNLA)
	if disableNLA {
		log.Printf("NLA disabled for this connection")
	}
	
	// Enable audio if requested
	if enableAudio {
		rdpClient.EnableAudio()
		log.Printf("Audio redirection enabled")
	}

	if err = rdpClient.Connect(); err != nil {
		log.Println(fmt.Errorf("rdp connect: %w", err))
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

	go wsToRdp(ctx, wsConn, rdpClient, cancel)
	rdpToWs(ctx, rdpClient, wsConn)
}

func wsToRdp(ctx context.Context, wsConn *websocket.Conn, rdpConn rdpConn, cancel context.CancelFunc) {
	defer cancel()

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
			log.Println(fmt.Errorf("error reading message from ws: %w", err))
			return
		}

		if err := rdpConn.SendInputEvent(data); err != nil {
			log.Println(fmt.Errorf("failed writing to rdp: %w", err))
			return
		}
	}
}

var wsMutex sync.Mutex

func rdpToWs(ctx context.Context, rdpConn rdpConn, wsConn *websocket.Conn) {
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
			log.Println(fmt.Errorf("get update: %w", err))
			return
		}

		wsMutex.Lock()
		err = websocket.Message.Send(wsConn, update.Data)
		wsMutex.Unlock()

		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			log.Println(fmt.Errorf("failed sending message to ws: %w", err))
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

	log.Printf("Server Capabilities: codecs=%v surfaceCmds=%v colorDepth=%d desktop=%s multifrag=%d largePtr=%v frameAck=%v",
		caps.BitmapCodecs, caps.SurfaceCommands, caps.ColorDepth, caps.DesktopSize,
		caps.MultifragmentSize, caps.LargePointer, caps.FrameAcknowledge)

	msg := buildCapabilitiesMessage(caps)

	wsMutex.Lock()
	defer wsMutex.Unlock()
	if err := websocket.Message.Send(wsConn, msg); err != nil {
		log.Printf("Failed to send capabilities info: %v", err)
	}
}

// buildCapabilitiesMessage creates the capabilities JSON message
func buildCapabilitiesMessage(caps *rdp.ServerCapabilityInfo) []byte {
	jsonData := fmt.Sprintf(`{"type":"capabilities","codecs":[%s],"surfaceCommands":%t,"colorDepth":%d,"desktopSize":"%s","multifragmentSize":%d,"largePointer":%t,"frameAcknowledge":%t}`,
		codecListToJSON(caps.BitmapCodecs),
		caps.SurfaceCommands,
		caps.ColorDepth,
		caps.DesktopSize,
		caps.MultifragmentSize,
		caps.LargePointer,
		caps.FrameAcknowledge)

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
		log.Printf("Failed to send audio data: %v", err)
	}
}
