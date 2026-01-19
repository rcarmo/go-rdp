package mcs

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/rcarmo/rdp-html5/internal/protocol/encoding"
)

type ClientSendDataRequest struct {
	Initiator uint16
	ChannelId uint16
	Data      []byte
}

func (d *ClientSendDataRequest) Serialize() []byte {
	buf := new(bytes.Buffer)

	encoding.PerWriteInteger16(d.Initiator, 1001, buf)
	encoding.PerWriteInteger16(d.ChannelId, 0, buf)
	buf.WriteByte(0x70) // magic word
	encoding.BerWriteLength(int(len(d.Data)), buf)

	buf.Write(d.Data)

	return buf.Bytes()
}

func (d *ClientSendDataRequest) Deserialize(wire io.Reader) error {
	var err error

	d.Initiator, err = encoding.PerReadInteger16(1001, wire)
	if err != nil {
		return err
	}

	d.ChannelId, err = encoding.PerReadInteger16(0, wire)
	if err != nil {
		return err
	}

	var magic uint8
	err = binary.Read(wire, binary.LittleEndian, &magic)
	if err != nil {
		return err
	}

	_, err = encoding.BerReadLength(wire)
	if err != nil {
		return err
	}

	return nil
}

func (p *Protocol) Send(userID, channelID uint16, pduData []byte) error {
	req := DomainPDU{
		Application: SendDataRequest,
		ClientSendDataRequest: &ClientSendDataRequest{
			Initiator: userID,
			ChannelId: channelID,
			Data:      pduData,
		},
	}

	if err := p.x224Conn.Send(req.Serialize()); err != nil {
		return fmt.Errorf("client MCS send data request: %w", err)
	}

	return nil
}
