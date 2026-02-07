// Package gcc implements Generic Conference Control (T.124) structures
// used in RDP connection sequence as specified in MS-RDPBCGR.
package gcc

import (
	"bytes"

	"github.com/rcarmo/go-rdp/internal/protocol/encoding"
)

var (
	t124_02_98_oid = [6]byte{0, 0, 20, 124, 0, 1}
	h221CSKey      = "Duca"
	h221SCKey      = "McDn"
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
	encoding.PerWriteLength(uint16(14+len(r.UserData)), buf) // #nosec G115

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
