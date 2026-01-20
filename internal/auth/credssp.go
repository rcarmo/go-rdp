package auth

import (
	"bytes"
	"crypto/sha256"
)

// TSRequest represents a decoded CredSSP request
type TSRequest struct {
	Version      int
	NegoTokens   []NegoToken
	AuthInfo     []byte
	PubKeyAuth   []byte
	ClientNonce  []byte // For version 5+
	ErrorCode    uint32 // For version 3+
	ServerNonce  []byte // Received from server in version 5+
}

// Magic strings for CredSSP version 5+ public key hashing (includes null terminator)
var (
	ClientServerHashMagic = []byte("CredSSP Client-To-Server Binding Hash\x00")
	ServerClientHashMagic = []byte("CredSSP Server-To-Client Binding Hash\x00")
)

// ComputeClientPubKeyAuth computes the pubKeyAuth for the client
// For version 2-4: just encrypt the public key
// For version 5+: compute SHA256(magic || nonce || pubKey) and encrypt
func ComputeClientPubKeyAuth(version int, pubKey, nonce []byte) []byte {
	if version >= 5 && len(nonce) > 0 {
		// Version 5+: Hash-based binding
		// Per FreeRDP: SHA256(ClientServerHashMagic || ClientNonce || SubjectPublicKey)
		h := sha256.New()
		h.Write(ClientServerHashMagic)
		h.Write(nonce)
		h.Write(pubKey)
		return h.Sum(nil)
	}
	// Version 2-4: Direct public key (will be encrypted by caller)
	return pubKey
}

// VerifyServerPubKeyAuth verifies the server's pubKeyAuth response
// For version 2-4: server sends pubKey with first byte incremented by 1
// For version 5+: server sends SHA256(ServerClientHashMagic || nonce || pubKey)
func VerifyServerPubKeyAuth(version int, serverPubKeyAuth, clientPubKey, nonce []byte) bool {
	if version >= 5 && len(nonce) > 0 {
		// Version 5+: Hash-based verification
		h := sha256.New()
		h.Write(ServerClientHashMagic)
		h.Write(nonce)
		h.Write(clientPubKey)
		expected := h.Sum(nil)
		return bytes.Equal(serverPubKeyAuth, expected)
	}
	// Version 2-4: Server sends pubKey with first byte + 1
	if len(serverPubKeyAuth) != len(clientPubKey) {
		return false
	}
	expected := make([]byte, len(clientPubKey))
	copy(expected, clientPubKey)
	expected[0]++
	return bytes.Equal(serverPubKeyAuth, expected)
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
//	   errorCode  [4] INTEGER OPTIONAL,       -- version 3+
//	   clientNonce [5] OCTET STRING OPTIONAL, -- version 5+
//	}
//	NegoData ::= SEQUENCE OF NegoDataItem
//	NegoDataItem ::= SEQUENCE {
//	   negoToken [0] OCTET STRING
//	}
func EncodeTSRequest(ntlmMessages [][]byte, authInfo []byte, pubKeyAuth []byte) []byte {
	return EncodeTSRequestWithNonce(ntlmMessages, authInfo, pubKeyAuth, nil)
}

// EncodeTSRequestWithNonce encodes a TSRequest with optional client nonce (version 5+)
// The version parameter should be set by the caller to match the negotiated version
func EncodeTSRequestWithNonce(ntlmMessages [][]byte, authInfo []byte, pubKeyAuth []byte, clientNonce []byte) []byte {
	return EncodeTSRequestWithVersion(6, ntlmMessages, authInfo, pubKeyAuth, clientNonce)
}

// EncodeTSRequestWithVersion encodes a TSRequest with explicit version control
func EncodeTSRequestWithVersion(version int, ntlmMessages [][]byte, authInfo []byte, pubKeyAuth []byte, clientNonce []byte) []byte {
	buf := &bytes.Buffer{}

	// Build the inner content first
	inner := &bytes.Buffer{}

	// [0] version INTEGER
	versionBytes := encodeContextTag(0, encodeInteger(version))
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

	// [5] clientNonce OCTET STRING OPTIONAL (version 5+)
	if len(clientNonce) > 0 {
		clientNonceBytes := encodeContextTag(5, encodeOctetString(clientNonce))
		inner.Write(clientNonceBytes)
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
		case 4: // errorCode (version 3+)
			req.ErrorCode = uint32(parseInteger(value))
		case 5: // clientNonce/serverNonce (version 5+)
			_, inner, _ := parseTag(value)
			req.ServerNonce = inner
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
