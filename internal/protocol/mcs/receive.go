package mcs

import (
	"io"

	"github.com/rcarmo/rdp-html5/internal/protocol/encoding"
)

type ServerSendDataIndication struct {
	Initiator uint16
	ChannelId uint16
}

func (d *ServerSendDataIndication) Deserialize(wire io.Reader) error {
	var err error

	d.Initiator, err = encoding.PerReadInteger16(1001, wire)
	if err != nil {
		return err
	}

	d.ChannelId, err = encoding.PerReadInteger16(0, wire)
	if err != nil {
		return err
	}

	_, err = encoding.PerReadEnumerates(wire)
	if err != nil {
		return err
	}

	_, err = encoding.BerReadLength(wire)
	if err != nil {
		return err
	}

	return nil
}

// Receive returns channelName, reader or error
func (p *Protocol) Receive() (uint16, io.Reader, error) {
	wire, err := p.x224Conn.Receive()
	if err != nil {
		return 0, nil, err
	}

	var resp DomainPDU
	if err = resp.Deserialize(wire); err != nil {
		return 0, nil, err
	}

	if resp.Application != SendDataIndication {
		// Log what we actually received for debugging
		if resp.Application == disconnectProviderUltimatum {
			return 0, nil, ErrDisconnectUltimatum
		}
		return 0, nil, ErrUnknownDomainApplication
	}

	return resp.ServerSendDataIndication.ChannelId, wire, nil
}
