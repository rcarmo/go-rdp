// Package rdp exposes the embeddable RDP client API.
package rdp

import (
	internallogging "github.com/rcarmo/go-rdp/internal/logging"
	internal "github.com/rcarmo/go-rdp/internal/rdp"
)

type (
	Client               = internal.Client
	Update               = internal.Update
	ServerCapabilityInfo = internal.ServerCapabilityInfo
	RemoteApp            = internal.RemoteApp
	RailState            = internal.RailState
	AudioHandler         = internal.AudioHandler
)

var ErrUnsupportedRequestedProtocol = internal.ErrUnsupportedRequestedProtocol

var NewClient = internal.NewClient
var NewClientWithDialContext = internal.NewClientWithDialContext

// SetLogLevel sets the library's log verbosity: "debug", "info", "warn", or
// "error". The library logs to stderr at info level by default.
func SetLogLevel(level string) {
	internallogging.Default().SetLevelFromString(level)
}
