package mcs

import "io"

type x224Conn interface {
	Receive() (io.Reader, error)
	Send(pduData []byte) error
}

// MCSLayer defines the interface for MCS protocol operations.
// This allows for mocking in tests.
type MCSLayer interface {
	// Send sends data through MCS layer
	Send(userID, channelID uint16, data []byte) error
	// Receive receives data from MCS layer
	Receive() (channelID uint16, reader io.Reader, err error)
	// Connect establishes MCS connection
	Connect(userData []byte) (io.Reader, error)
	// ErectDomain erects MCS domain
	ErectDomain() error
	// AttachUser attaches user to MCS
	AttachUser() (uint16, error)
	// JoinChannels joins MCS channels
	JoinChannels(userID uint16, channelIDMap map[string]uint16) error
}
