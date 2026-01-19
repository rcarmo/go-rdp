package gcc

import (
	"bytes"

	"github.com/rcarmo/rdp-html5/internal/protocol/encoding"
)

type ConferenceCreateRequest struct {
	UserData []byte
}

func NewConferenceCreateRequest(userData []byte) *ConferenceCreateRequest {
	return &ConferenceCreateRequest{
		UserData: userData,
	}
}

func (r *ConferenceCreateRequest) Serialize() []byte {
	buf := new(bytes.Buffer)

	encoding.PerWriteChoice(0, buf)
	encoding.PerWriteObjectIdentifier(t124_02_98_oid, buf)
	encoding.PerWriteLength(uint16(14+len(r.UserData)), buf)

	encoding.PerWriteChoice(0, buf)
	encoding.PerWriteSelection(0x08, buf)

	encoding.PerWriteNumericString("1", 1, buf)
	encoding.PerWritePadding(1, buf)
	encoding.PerWriteNumberOfSet(1, buf)
	encoding.PerWriteChoice(0xc0, buf)
	encoding.PerWriteOctetStream(h221CSKey, 4, buf)
	encoding.PerWriteOctetStream(string(r.UserData), 0, buf)

	return buf.Bytes()
}
