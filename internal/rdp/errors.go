package rdp

import "errors"

// ErrUnsupportedRequestedProtocol indicates that the server selected a protocol
// that this client does not support.
var (
	ErrUnsupportedRequestedProtocol = errors.New("unsupported requested protocol")
)
