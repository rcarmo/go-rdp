package mcs

import (
	"bytes"
	"fmt"

	"github.com/rcarmo/rdp-html5/internal/protocol/encoding"
)

type ClientErectDomainRequest struct{}

func (pdu *ClientErectDomainRequest) Serialize() []byte {
	buf := new(bytes.Buffer)

	encoding.PerWriteInteger(0, buf)
	encoding.PerWriteInteger(0, buf)

	return buf.Bytes()
}

func (p *Protocol) ErectDomain() error {
	req := DomainPDU{
		Application:              erectDomainRequest,
		ClientErectDomainRequest: &ClientErectDomainRequest{},
	}

	if err := p.x224Conn.Send(req.Serialize()); err != nil {
		return fmt.Errorf("client MCS erect domain request: %w", err)
	}

	return nil
}
