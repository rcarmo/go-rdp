package pdu

import "errors"

var (
	ErrInvalidCorrelationID = errors.New("invalid correlationId")
	ErrDeactivateAll         = errors.New("deactivate all")
)
