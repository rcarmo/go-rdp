package mcs

import (
	"bytes"
	"io"

	"github.com/rcarmo/go-rdp/internal/protocol/encoding"
)

const (
	RTSuccessful uint8 = iota
	RTDomainMerging
	RTDomainNotHierarchical
	RTNoSuchChannel
	RTNoSuchDomain
	RTNoSuchUser
	RTNotAdmitted
	RTOtherUserId
	RTParametersUnacceptable
	RTTokenNotAvailable
	RTTokenNotPossessed
	RTTooManyChannels
	RTTooManyTokens
	RTTooManyUsers
	RTUnspecifiedFailure
	RTUserRejected
)

const (
	RNDomainDisconnected uint8 = iota
	RNProviderInitiated
	RNTokenPurged
	RNUserRequested
	RNChannelPurged
)

type domainParameters struct {
	maxChannelIds   int
	maxUserIds      int
	maxTokenIds     int
	numPriorities   int
	minThroughput   int
	maxHeight       int
	maxMCSPDUsize   int
	protocolVersion int
}

func (params *domainParameters) Serialize() []byte {
	buf := new(bytes.Buffer)

	encoding.BerWriteInteger(params.maxChannelIds, buf)
	encoding.BerWriteInteger(params.maxUserIds, buf)
	encoding.BerWriteInteger(params.maxTokenIds, buf)
	encoding.BerWriteInteger(params.numPriorities, buf)
	encoding.BerWriteInteger(params.minThroughput, buf)
	encoding.BerWriteInteger(params.maxHeight, buf)
	encoding.BerWriteInteger(params.maxMCSPDUsize, buf)
	encoding.BerWriteInteger(params.protocolVersion, buf)

	return buf.Bytes()
}

func (params *domainParameters) Deserialize(wire io.Reader) error {
	var err error

	params.maxChannelIds, err = encoding.BerReadInteger(wire)
	if err != nil {
		return err
	}

	params.maxUserIds, err = encoding.BerReadInteger(wire)
	if err != nil {
		return err
	}

	params.maxTokenIds, err = encoding.BerReadInteger(wire)
	if err != nil {
		return err
	}

	params.numPriorities, err = encoding.BerReadInteger(wire)
	if err != nil {
		return err
	}

	params.minThroughput, err = encoding.BerReadInteger(wire)
	if err != nil {
		return err
	}

	params.maxHeight, err = encoding.BerReadInteger(wire)
	if err != nil {
		return err
	}

	params.maxMCSPDUsize, err = encoding.BerReadInteger(wire)
	if err != nil {
		return err
	}

	params.protocolVersion, err = encoding.BerReadInteger(wire)
	if err != nil {
		return err
	}

	return nil
}
