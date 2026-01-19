package auth

import (
	"bytes"
)

// TSRequest represents a decoded CredSSP request
type TSRequest struct {
	Version    int
	NegoTokens []NegoToken
	AuthInfo   []byte
	PubKeyAuth []byte
}

// NegoToken wraps an NTLM message
type NegoToken struct {
	Data []byte
}

// EncodeTSRequest encodes a TSRequest with NTLM messages, auth info, and/or public key auth
// Per MS-CSSP, TSRequest is:
//
//	TSRequest ::= SEQUENCE {
//	   version    [0] INTEGER,
//	   negoTokens [1] NegoData OPTIONAL,
//	   authInfo   [2] OCTET STRING OPTIONAL,
//	   pubKeyAuth [3] OCTET STRING OPTIONAL,
//	}
//	NegoData ::= SEQUENCE OF NegoDataItem
//	NegoDataItem ::= SEQUENCE {
//	   negoToken [0] OCTET STRING
//	}
func EncodeTSRequest(ntlmMessages [][]byte, authInfo []byte, pubKeyAuth []byte) []byte {
	buf := &bytes.Buffer{}

	// Build the inner content first
	inner := &bytes.Buffer{}

	// [0] version INTEGER
	versionBytes := encodeContextTag(0, encodeInteger(2))
	inner.Write(versionBytes)

	// [1] negoTokens NegoData OPTIONAL
	if len(ntlmMessages) > 0 {
		negoData := &bytes.Buffer{}
		for _, msg := range ntlmMessages {
			// NegoDataItem ::= SEQUENCE { negoToken [0] OCTET STRING }
			item := encodeSequence(encodeContextTag(0, encodeOctetString(msg)))
			negoData.Write(item)
		}
		// NegoData ::= SEQUENCE OF NegoDataItem
		negoTokens := encodeContextTag(1, encodeSequence(negoData.Bytes()))
		inner.Write(negoTokens)
	}

	// [2] authInfo OCTET STRING OPTIONAL
	if len(authInfo) > 0 {
		authInfoBytes := encodeContextTag(2, encodeOctetString(authInfo))
		inner.Write(authInfoBytes)
	}

	// [3] pubKeyAuth OCTET STRING OPTIONAL
	if len(pubKeyAuth) > 0 {
		pubKeyAuthBytes := encodeContextTag(3, encodeOctetString(pubKeyAuth))
		inner.Write(pubKeyAuthBytes)
	}

	// Wrap in SEQUENCE
	buf.Write(encodeSequence(inner.Bytes()))
	return buf.Bytes()
}

// DecodeTSRequest decodes a TSRequest from DER bytes
func DecodeTSRequest(data []byte) (*TSRequest, error) {
	req := &TSRequest{}

	// Parse outer SEQUENCE
	_, content, err := parseTag(data)
	if err != nil {
		return nil, err
	}

	// Parse fields
	offset := 0
	for offset < len(content) {
		tag, value, err := parseTag(content[offset:])
		if err != nil {
			break
		}
		ctxTag := tag & 0x1F

		switch ctxTag {
		case 0: // version
			req.Version = parseInteger(value)
		case 1: // negoTokens
			req.NegoTokens = parseNegoTokens(value)
		case 2: // authInfo
			_, inner, _ := parseTag(value)
			req.AuthInfo = inner
		case 3: // pubKeyAuth
			_, inner, _ := parseTag(value)
			req.PubKeyAuth = inner
		}

		offset += tagLen(content[offset:])
	}

	return req, nil
}

// EncodeCredentials encodes TSCredentials with password authentication
//
//	TSCredentials ::= SEQUENCE {
//	   credType    [0] INTEGER,
//	   credentials [1] OCTET STRING
//	}
//	TSPasswordCreds ::= SEQUENCE {
//	   domainName [0] OCTET STRING,
//	   userName   [1] OCTET STRING,
//	   password   [2] OCTET STRING
//	}
func EncodeCredentials(domain, username, password []byte) []byte {
	// Encode TSPasswordCreds
	passCreds := &bytes.Buffer{}
	passCreds.Write(encodeContextTag(0, encodeOctetString(domain)))
	passCreds.Write(encodeContextTag(1, encodeOctetString(username)))
	passCreds.Write(encodeContextTag(2, encodeOctetString(password)))
	passCredsSeq := encodeSequence(passCreds.Bytes())

	// Encode TSCredentials
	creds := &bytes.Buffer{}
	creds.Write(encodeContextTag(0, encodeInteger(1))) // credType = 1 (password)
	creds.Write(encodeContextTag(1, encodeOctetString(passCredsSeq)))

	return encodeSequence(creds.Bytes())
}

// ASN.1 DER encoding helpers

func encodeLength(length int) []byte {
	if length < 128 {
		return []byte{byte(length)}
	}
	if length < 256 {
		return []byte{0x81, byte(length)}
	}
	if length < 65536 {
		return []byte{0x82, byte(length >> 8), byte(length)}
	}
	return []byte{0x83, byte(length >> 16), byte(length >> 8), byte(length)}
}

func encodeSequence(data []byte) []byte {
	buf := &bytes.Buffer{}
	buf.WriteByte(0x30) // SEQUENCE tag
	buf.Write(encodeLength(len(data)))
	buf.Write(data)
	return buf.Bytes()
}

func encodeContextTag(tag int, data []byte) []byte {
	buf := &bytes.Buffer{}
	buf.WriteByte(0xA0 | byte(tag)) // Context-specific constructed tag
	buf.Write(encodeLength(len(data)))
	buf.Write(data)
	return buf.Bytes()
}

func encodeOctetString(data []byte) []byte {
	buf := &bytes.Buffer{}
	buf.WriteByte(0x04) // OCTET STRING tag
	buf.Write(encodeLength(len(data)))
	buf.Write(data)
	return buf.Bytes()
}

func encodeInteger(val int) []byte {
	buf := &bytes.Buffer{}
	buf.WriteByte(0x02) // INTEGER tag
	if val < 128 {
		buf.WriteByte(1)
		buf.WriteByte(byte(val))
	} else if val < 256 {
		buf.WriteByte(2)
		buf.WriteByte(0)
		buf.WriteByte(byte(val))
	} else {
		buf.WriteByte(2)
		buf.WriteByte(byte(val >> 8))
		buf.WriteByte(byte(val))
	}
	return buf.Bytes()
}

// DER parsing helpers

func parseTag(data []byte) (byte, []byte, error) {
	if len(data) < 2 {
		return 0, nil, bytes.ErrTooLarge
	}
	tag := data[0]
	lenByte := data[1]
	offset := 2
	var length int

	if lenByte < 128 {
		length = int(lenByte)
	} else {
		numBytes := int(lenByte & 0x7F)
		if offset+numBytes > len(data) {
			return 0, nil, bytes.ErrTooLarge
		}
		for i := 0; i < numBytes; i++ {
			length = (length << 8) | int(data[offset])
			offset++
		}
	}

	if offset+length > len(data) {
		return 0, nil, bytes.ErrTooLarge
	}
	return tag, data[offset : offset+length], nil
}

func tagLen(data []byte) int {
	if len(data) < 2 {
		return len(data)
	}
	lenByte := data[1]
	offset := 2
	var length int

	if lenByte < 128 {
		length = int(lenByte)
	} else {
		numBytes := int(lenByte & 0x7F)
		offset += numBytes
		for i := 0; i < numBytes && 2+i < len(data); i++ {
			length = (length << 8) | int(data[2+i])
		}
	}
	return offset + length
}

func parseInteger(data []byte) int {
	_, value, err := parseTag(data)
	if err != nil || len(value) == 0 {
		return 0
	}
	result := 0
	for _, b := range value {
		result = (result << 8) | int(b)
	}
	return result
}

func parseNegoTokens(data []byte) []NegoToken {
	var tokens []NegoToken

	// Parse outer SEQUENCE
	_, content, err := parseTag(data)
	if err != nil {
		return tokens
	}

	// Parse each NegoDataItem
	offset := 0
	for offset < len(content) {
		// SEQUENCE (NegoDataItem)
		_, item, err := parseTag(content[offset:])
		if err != nil {
			break
		}

		// [0] negoToken
		_, tokenData, err := parseTag(item)
		if err == nil {
			_, octetStr, err := parseTag(tokenData)
			if err == nil {
				tokens = append(tokens, NegoToken{Data: octetStr})
			}
		}

		offset += tagLen(content[offset:])
	}

	return tokens
}
