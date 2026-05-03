package auth

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rc4" // #nosec G505 -- RC4 is required by NTLMv2 authentication protocol
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"unicode/utf16"
)

const (
	ntlmMessageNegotiate    = 1
	ntlmMessageChallenge    = 2
	ntlmMessageAuthenticate = 3
)

// NTLMNegotiateMessage represents an NTLM Type 1 negotiate message.
type NTLMNegotiateMessage struct {
	Flags       uint32
	Domain      string
	Workstation string
	RawData     []byte
}

// NTLMAuthenticateMessage represents an NTLM Type 3 authenticate message.
type NTLMAuthenticateMessage struct {
	Flags                     uint32
	Domain                    string
	User                      string
	Workstation               string
	LMResponse                []byte
	NTResponse                []byte
	EncryptedRandomSessionKey []byte
	MIC                       []byte
	RawData                   []byte
}

// ServerNTLMv2 handles the server side of NTLMv2 authentication for CredSSP/NLA.
type ServerNTLMv2 struct {
	Domain       string
	Computer     string
	Challenge    [8]byte
	TargetInfo   []byte
	Negotiate    []byte
	ChallengePDU []byte
}

// NewServerNTLMv2 creates a server-side NTLMv2 context.
func NewServerNTLMv2(domain, computer string) (*ServerNTLMv2, error) {
	if computer == "" {
		computer = "GO-RDP"
	}
	s := &ServerNTLMv2{Domain: domain, Computer: computer}
	if _, err := rand.Read(s.Challenge[:]); err != nil {
		return nil, err
	}
	s.TargetInfo = BuildTargetInfo(domain, computer)
	return s, nil
}

// BuildChallengeMessage processes an NTLM Type 1 message and returns a Type 2 challenge.
func (s *ServerNTLMv2) BuildChallengeMessage(negotiateData []byte) ([]byte, error) {
	nego, err := ParseNegotiateMessage(negotiateData)
	if err != nil {
		return nil, err
	}
	s.Negotiate = append([]byte(nil), negotiateData...)

	flags := nego.Flags & (NTLMSSP_NEGOTIATE_UNICODE | NTLMSSP_NEGOTIATE_SIGN | NTLMSSP_NEGOTIATE_SEAL | NTLMSSP_NEGOTIATE_128 | NTLMSSP_NEGOTIATE_56 | NTLMSSP_NEGOTIATE_KEY_EXCH)
	flags |= NTLMSSP_REQUEST_TARGET |
		NTLMSSP_NEGOTIATE_TARGET_INFO |
		NTLMSSP_NEGOTIATE_EXTENDED_SESSIONSECURITY |
		NTLMSSP_NEGOTIATE_ALWAYS_SIGN |
		NTLMSSP_NEGOTIATE_NTLM |
		NTLMSSP_NEGOTIATE_VERSION |
		NTLMSSP_TARGET_TYPE_SERVER
	if flags&NTLMSSP_NEGOTIATE_UNICODE == 0 {
		flags |= NTLMSSP_NEGOTIATE_UNICODE
	}

	targetName := unicodeEncode(s.Computer)
	targetNameOffset := uint32(56)
	targetInfoOffset := targetNameOffset + uint32(len(targetName)) // #nosec G115

	buf := &bytes.Buffer{}
	buf.Write(ntlmSignature)
	_ = binary.Write(buf, binary.LittleEndian, uint32(ntlmMessageChallenge))
	writeSecurityBuffer(buf, len(targetName), targetNameOffset)
	_ = binary.Write(buf, binary.LittleEndian, flags)
	buf.Write(s.Challenge[:])
	buf.Write(make([]byte, 8))
	writeSecurityBuffer(buf, len(s.TargetInfo), targetInfoOffset)
	buf.Write([]byte{0x06, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0f})
	buf.Write(targetName)
	buf.Write(s.TargetInfo)

	s.ChallengePDU = buf.Bytes()
	return append([]byte(nil), s.ChallengePDU...), nil
}

// VerifyAuthenticateMessage validates an NTLM Type 3 message using NTLMv2 and returns a sealing context.
func (s *ServerNTLMv2) VerifyAuthenticateMessage(authData []byte, username, password, domain string) (*NTLMAuthenticateMessage, *Security, error) {
	msg, err := ParseAuthenticateMessage(authData)
	if err != nil {
		return nil, nil, err
	}
	if username != "" && msg.User != username {
		return nil, nil, fmt.Errorf("unexpected NTLM user %q", msg.User)
	}
	if domain == "" {
		domain = msg.Domain
	}
	if len(msg.NTResponse) < 16+28 {
		return nil, nil, fmt.Errorf("NTLMv2 response too short")
	}

	ntProof := msg.NTResponse[:16]
	temp := msg.NTResponse[16:]
	respKeyNT := ntowfv2(password, msg.User, domain)
	expectedProof := hmacMD5(respKeyNT, append(s.Challenge[:], temp...))
	if !hmac.Equal(ntProof, expectedProof) {
		return nil, nil, fmt.Errorf("NTLMv2 proof mismatch")
	}
	sessionBaseKey := hmacMD5(respKeyNT, ntProof)
	exportedSessionKey := sessionBaseKey
	if len(msg.EncryptedRandomSessionKey) > 0 && msg.Flags&NTLMSSP_NEGOTIATE_KEY_EXCH != 0 {
		exportedSessionKey = make([]byte, len(msg.EncryptedRandomSessionKey))
		rc, err := rc4.NewCipher(sessionBaseKey) // #nosec G405 -- RC4 is required by NTLMv2 authentication protocol
		if err != nil {
			return nil, nil, err
		}
		rc.XORKeyStream(exportedSessionKey, msg.EncryptedRandomSessionKey)
	}

	if len(msg.MIC) == 16 && !allZero(msg.MIC) && len(s.Negotiate) > 0 && len(s.ChallengePDU) > 0 {
		micZeroed := append([]byte(nil), authData...)
		for i := 72; i < 88 && i < len(micZeroed); i++ {
			micZeroed[i] = 0
		}
		micInput := append(append(append([]byte(nil), s.Negotiate...), s.ChallengePDU...), micZeroed...)
		expectedMIC := hmacMD5(exportedSessionKey, micInput)[:16]
		if !hmac.Equal(msg.MIC, expectedMIC) {
			return nil, nil, fmt.Errorf("NTLM MIC mismatch")
		}
	}

	clientSigningKey := md5Hash(append(exportedSessionKey, append([]byte("session key to client-to-server signing key magic constant"), 0x00)...))
	serverSigningKey := md5Hash(append(exportedSessionKey, append([]byte("session key to server-to-client signing key magic constant"), 0x00)...))
	clientSealingKey := md5Hash(append(exportedSessionKey, append([]byte("session key to client-to-server sealing key magic constant"), 0x00)...))
	serverSealingKey := md5Hash(append(exportedSessionKey, append([]byte("session key to server-to-client sealing key magic constant"), 0x00)...))

	decryptRC4, _ := rc4.NewCipher(clientSealingKey) // #nosec G405 -- RC4 is required by NTLMv2 authentication protocol
	encryptRC4, _ := rc4.NewCipher(serverSealingKey) // #nosec G405 -- RC4 is required by NTLMv2 authentication protocol
	return msg, &Security{encryptRC4: encryptRC4, decryptRC4: decryptRC4, signingKey: serverSigningKey, verifyKey: clientSigningKey}, nil
}

// ParseNegotiateMessage parses an NTLM Type 1 message.
func ParseNegotiateMessage(data []byte) (*NTLMNegotiateMessage, error) {
	if len(data) < 32 || !bytes.Equal(data[:8], ntlmSignature) {
		return nil, fmt.Errorf("invalid NTLM negotiate message")
	}
	if binary.LittleEndian.Uint32(data[8:12]) != ntlmMessageNegotiate {
		return nil, fmt.Errorf("not an NTLM negotiate message")
	}
	msg := &NTLMNegotiateMessage{Flags: binary.LittleEndian.Uint32(data[12:16]), RawData: append([]byte(nil), data...)}
	msg.Domain = decodeSecurityBufferString(data, 16, msg.Flags)
	msg.Workstation = decodeSecurityBufferString(data, 24, msg.Flags)
	return msg, nil
}

// ParseAuthenticateMessage parses an NTLM Type 3 authenticate message.
func ParseAuthenticateMessage(data []byte) (*NTLMAuthenticateMessage, error) {
	if len(data) < 64 || !bytes.Equal(data[:8], ntlmSignature) {
		return nil, fmt.Errorf("invalid NTLM authenticate message")
	}
	if binary.LittleEndian.Uint32(data[8:12]) != ntlmMessageAuthenticate {
		return nil, fmt.Errorf("not an NTLM authenticate message")
	}
	msg := &NTLMAuthenticateMessage{RawData: append([]byte(nil), data...)}
	var err error
	if msg.LMResponse, err = readSecurityBuffer(data, 12); err != nil {
		return nil, err
	}
	if msg.NTResponse, err = readSecurityBuffer(data, 20); err != nil {
		return nil, err
	}
	msg.Domain = decodeSecurityBufferString(data, 28, NTLMSSP_NEGOTIATE_UNICODE)
	msg.User = decodeSecurityBufferString(data, 36, NTLMSSP_NEGOTIATE_UNICODE)
	msg.Workstation = decodeSecurityBufferString(data, 44, NTLMSSP_NEGOTIATE_UNICODE)
	if msg.EncryptedRandomSessionKey, err = readSecurityBuffer(data, 52); err != nil {
		return nil, err
	}
	if len(data) >= 64 {
		msg.Flags = binary.LittleEndian.Uint32(data[60:64])
	}
	if len(data) >= 88 {
		msg.MIC = append([]byte(nil), data[72:88]...)
	}
	return msg, nil
}

// BuildTargetInfo builds NTLM AV pairs for a server challenge.
func BuildTargetInfo(domain, computer string) []byte {
	buf := &bytes.Buffer{}
	writeAVPairString(buf, MsvAvNbComputerName, computer)
	if domain != "" {
		writeAVPairString(buf, MsvAvNbDomainName, domain)
	}
	writeAVPairString(buf, MsvAvDnsComputerName, computer)
	if domain != "" {
		writeAVPairString(buf, MsvAvDnsDomainName, domain)
	}
	writeAVPairBytes(buf, MsvAvTimestamp, makeTimestamp())
	_ = binary.Write(buf, binary.LittleEndian, uint16(MsvAvEOL))
	_ = binary.Write(buf, binary.LittleEndian, uint16(0))
	return buf.Bytes()
}

// ComputeServerPubKeyAuth computes the server-to-client CredSSP pubKeyAuth response.
func ComputeServerPubKeyAuth(version int, pubKey, nonce []byte) []byte {
	if version >= 5 && len(nonce) > 0 {
		h := sha256.New()
		h.Write(ServerClientHashMagic)
		h.Write(nonce)
		h.Write(pubKey)
		return h.Sum(nil)
	}
	result := append([]byte(nil), pubKey...)
	if len(result) > 0 {
		result[0]++
	}
	return result
}

func writeSecurityBuffer(buf *bytes.Buffer, length int, offset uint32) {
	_ = binary.Write(buf, binary.LittleEndian, uint16(length)) // #nosec G115
	_ = binary.Write(buf, binary.LittleEndian, uint16(length)) // #nosec G115
	_ = binary.Write(buf, binary.LittleEndian, offset)
}

func readSecurityBuffer(data []byte, off int) ([]byte, error) {
	if off+8 > len(data) {
		return nil, fmt.Errorf("security buffer header exceeds message")
	}
	length := int(binary.LittleEndian.Uint16(data[off : off+2]))
	bufOff := int(binary.LittleEndian.Uint32(data[off+4 : off+8]))
	if length == 0 {
		return nil, nil
	}
	if bufOff < 0 || bufOff+length > len(data) {
		return nil, fmt.Errorf("security buffer exceeds message")
	}
	return append([]byte(nil), data[bufOff:bufOff+length]...), nil
}

func decodeSecurityBufferString(data []byte, off int, flags uint32) string {
	buf, err := readSecurityBuffer(data, off)
	if err != nil || len(buf) == 0 {
		return ""
	}
	if flags&NTLMSSP_NEGOTIATE_UNICODE != 0 {
		return unicodeDecode(buf)
	}
	return string(buf)
}

func unicodeDecode(data []byte) string {
	if len(data)%2 != 0 {
		data = data[:len(data)-1]
	}
	runes := make([]uint16, len(data)/2)
	for i := range runes {
		runes[i] = binary.LittleEndian.Uint16(data[i*2:])
	}
	return string(utf16.Decode(runes))
}

func writeAVPairString(buf *bytes.Buffer, id uint16, value string) {
	writeAVPairBytes(buf, id, unicodeEncode(value))
}

func writeAVPairBytes(buf *bytes.Buffer, id uint16, value []byte) {
	_ = binary.Write(buf, binary.LittleEndian, id)
	_ = binary.Write(buf, binary.LittleEndian, uint16(len(value))) // #nosec G115
	buf.Write(value)
}

func allZero(data []byte) bool {
	for _, b := range data {
		if b != 0 {
			return false
		}
	}
	return true
}
