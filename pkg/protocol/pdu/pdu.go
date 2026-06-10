// Package pdu exposes selected RDP PDU errors and types used by embedders.
package pdu

import internal "github.com/rcarmo/go-rdp/internal/protocol/pdu"

type CapabilitySet = internal.CapabilitySet

var (
	ErrInvalidCorrelationID = internal.ErrInvalidCorrelationID
	ErrDeactivateAll        = internal.ErrDeactivateAll
	NSCodecGUID             = internal.NSCodecGUID
	RemoteFXImageGUID       = internal.RemoteFXImageGUID
)

var NewBitmapCodecsWithRFXCapabilitySet = internal.NewBitmapCodecsWithRFXCapabilitySet
