package pdu

import "errors"

var (
	// ErrInvalidCorrelationID indicates the correlation ID in the response does not match the request.
	ErrInvalidCorrelationID = errors.New("invalid correlationId")
	// ErrDeactivateAll indicates the server sent a Deactivate All PDU (MS-RDPBCGR 2.2.3.1).
	ErrDeactivateAll = errors.New("deactivate all")
)
