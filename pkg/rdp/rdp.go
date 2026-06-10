// Package rdp exposes the embeddable RDP client API.
package rdp

import internal "github.com/rcarmo/go-rdp/internal/rdp"

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
