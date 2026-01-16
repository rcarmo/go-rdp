package handler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"

	"github.com/kulaginds/rdp-html5/internal/pkg/config"
	"github.com/kulaginds/rdp-html5/internal/pkg/rdp"
	"github.com/kulaginds/rdp-html5/internal/pkg/rdp/fastpath"
	"github.com/kulaginds/rdp-html5/internal/pkg/rdp/pdu"
)

const (
	webSocketReadBufferSize  = 8192
	webSocketWriteBufferSize = 8192 * 2
)

type rdpConn interface {
	GetUpdate() (*fastpath.UpdatePDU, error)
	SendInputEvent(data []byte) error
}

func Connect(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  webSocketReadBufferSize,
		WriteBufferSize: webSocketWriteBufferSize,
		CheckOrigin: func(r *http.Request) bool {
			return isAllowedOrigin(r.Header.Get("Origin"))
		},
	}
	protocol := r.Header.Get("Sec-Websocket-Protocol")

	wsConn, err := upgrader.Upgrade(w, r, http.Header{
		"Sec-Websocket-Protocol": {protocol},
	})
	if err != nil {
		log.Println(fmt.Errorf("upgrade websocket: %w", err))

		return
	}

	defer func() {
		if err = wsConn.Close(); err != nil {
			log.Println(fmt.Errorf("error closing websocket: %w", err))
		}
	}()

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

	host := r.URL.Query().Get("host")
	user := r.URL.Query().Get("user")
	password := r.URL.Query().Get("password")

	rdpClient, err := rdp.NewClient(host, user, password, width, height)
	if err != nil {
		log.Println(fmt.Errorf("rdp init: %w", err))

		return
	}
	defer rdpClient.Close()

	// Set TLS configuration from server config (use global config if available)
	cfg := config.GetGlobalConfig()
	if cfg == nil {
		// Fallback to loading config if global config not available (for testing)
		var err error
		cfg, err = config.Load()
		if err != nil {
			log.Printf("Failed to load config for TLS settings: %v", err)
			cfg = &config.Config{}
		}
	}

	rdpClient.SetTLSConfig(cfg.Security.SkipTLSValidation, cfg.Security.TLSServerName)

	// Set NLA configuration - enable NLA by default for servers that require it
	rdpClient.SetUseNLA(cfg.Security.UseNLA)

	// TODO: implement
	//rdpClient.SetRemoteApp("C:\\agent\\agent.exe", ".\\Downloads\\cbct1.zip", "C:\\Users\\Doc")
	//rdpClient.SetRemoteApp("explore", "", "")

	if err = rdpClient.Connect(); err != nil {
		log.Println(fmt.Errorf("rdp connect: %w", err))

		return
	}

	go wsToRdp(ctx, wsConn, rdpClient, cancel)
	rdpToWs(ctx, rdpClient, wsConn)
}

func wsToRdp(ctx context.Context, wsConn *websocket.Conn, rdpConn rdpConn, cancel context.CancelFunc) {
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return
		default: // pass
		}

		_, data, err := wsConn.ReadMessage()
		if err != nil {
			if strings.HasSuffix(err.Error(), "use of closed network connection") {
				return
			}

			log.Println(fmt.Errorf("error reading message from ws: %w", err))

			return
		}

		if err = rdpConn.SendInputEvent(data); err != nil {
			log.Println(fmt.Errorf("failed writing to rdp: %w", err))

			return
		}
	}
}

func rdpToWs(ctx context.Context, rdpConn rdpConn, wsConn *websocket.Conn) {
	var (
		update *fastpath.UpdatePDU
		err    error
	)

	for {
		select {
		case <-ctx.Done():
			return
		default: // pass
		}

		update, err = rdpConn.GetUpdate()
		switch {
		case err == nil: // pass
		case errors.Is(err, pdu.ErrDeactiateAll):
			return

		default:
			log.Println(fmt.Errorf("get update: %w", err))

			return
		}

		if err = wsConn.WriteMessage(websocket.BinaryMessage, update.Data); err != nil {
			if err == websocket.ErrCloseSent {
				return
			}

			log.Println(fmt.Errorf("failed sending message to ws: %w", err))

			return
		}
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
		return strings.HasPrefix(normalized, "localhost") || strings.HasPrefix(normalized, "127.0.0.1")
	}

	// Always allow localhost-style origins for development, even when a list is provided
	if strings.HasPrefix(normalized, "localhost") || strings.HasPrefix(normalized, "127.0.0.1") {
		return true
	}

	for _, entry := range strings.Split(allowed, ",") {
		candidate := strings.TrimSpace(entry)
		if candidate == "" {
			continue
		}

		// Support allow-list entries with or without scheme
		if candidate == origin || candidate == normalized {
			return true
		}

		if strings.TrimPrefix(candidate, "http://") == normalized || strings.TrimPrefix(candidate, "https://") == normalized {
			return true
		}
	}

	return false
}
