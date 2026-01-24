package gcc

import (
	"errors"
	"io"

	"github.com/rcarmo/go-rdp/internal/protocol/encoding"
)

type ConferenceCreateResponse struct{}

func (r *ConferenceCreateResponse) Deserialize(wire io.Reader) error {
	_, err := encoding.PerReadChoice(wire)
	if err != nil {
		return err
	}

	var objectIdentifier bool

	objectIdentifier, err = encoding.PerReadObjectIdentifier(t124_02_98_oid, wire)
	if err != nil {
		return err
	}

	if !objectIdentifier {
		return errors.New("bad object identifier t124")
	}

	_, err = encoding.PerReadLength(wire)
	if err != nil {
		return err
	}

	_, err = encoding.PerReadChoice(wire)
	if err != nil {
		return err
	}

	_, err = encoding.PerReadInteger16(1001, wire)
	if err != nil {
		return err
	}

	_, err = encoding.PerReadInteger(wire)
	if err != nil {
		return err
	}

	_, err = encoding.PerReadEnumerates(wire)
	if err != nil {
		return err
	}

	_, err = encoding.PerReadNumberOfSet(wire)
	if err != nil {
		return err
	}

	_, err = encoding.PerReadChoice(wire)
	if err != nil {
		return err
	}

	var octetStream bool

	octetStream, err = encoding.PerReadOctetStream([]byte(h221SCKey), 4, wire)
	if err != nil {
		return err
	}

	if !octetStream {
		return errors.New("bad H221 SC_KEY")
	}

	_, err = encoding.PerReadLength(wire)
	if err != nil {
		return err
	}

	return nil
}
