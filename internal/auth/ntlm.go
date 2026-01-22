package auth

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/rc4"
	"encoding/binary"
	"time"
	"unicode/utf16"
)

// NTLM negotiate flags
const (
	NTLMSSP_NEGOTIATE_56                       = 0x80000000
	NTLMSSP_NEGOTIATE_KEY_EXCH                 = 0x40000000
	NTLMSSP_NEGOTIATE_128                      = 0x20000000
	NTLMSSP_NEGOTIATE_VERSION                  = 0x02000000
	NTLMSSP_NEGOTIATE_TARGET_INFO              = 0x00800000
	NTLMSSP_REQUEST_NON_NT_SESSION_KEY         = 0x00400000
	NTLMSSP_NEGOTIATE_IDENTIFY                 = 0x00100000
	NTLMSSP_NEGOTIATE_EXTENDED_SESSIONSECURITY = 0x00080000
	NTLMSSP_TARGET_TYPE_SERVER                 = 0x00020000
	NTLMSSP_TARGET_TYPE_DOMAIN                 = 0x00010000
	NTLMSSP_NEGOTIATE_ALWAYS_SIGN              = 0x00008000
	NTLMSSP_NEGOTIATE_OEM_WORKSTATION_SUPPLIED = 0x00002000
	NTLMSSP_NEGOTIATE_OEM_DOMAIN_SUPPLIED      = 0x00001000
	NTLMSSP_NEGOTIATE_NTLM                     = 0x00000200
	NTLMSSP_NEGOTIATE_LM_KEY                   = 0x00000080
	NTLMSSP_NEGOTIATE_DATAGRAM                 = 0x00000040
	NTLMSSP_NEGOTIATE_SEAL                     = 0x00000020
	NTLMSSP_NEGOTIATE_SIGN                     = 0x00000010
	NTLMSSP_REQUEST_TARGET                     = 0x00000004
	NTLM_NEGOTIATE_OEM                         = 0x00000002
	NTLMSSP_NEGOTIATE_UNICODE                  = 0x00000001
)

// AV Pair IDs
const (
	MsvAvEOL             = 0x0000
	MsvAvNbComputerName  = 0x0001
	MsvAvNbDomainName    = 0x0002
	MsvAvDnsComputerName = 0x0003
	MsvAvDnsDomainName   = 0x0004
	MsvAvDnsTreeName     = 0x0005
	MsvAvFlags           = 0x0006
	MsvAvTimestamp       = 0x0007
)

var ntlmSignature = []byte{'N', 'T', 'L', 'M', 'S', 'S', 'P', 0x00}

// NTLMv2 handles NTLMv2 authentication
type NTLMv2 struct {
	domain        string
	user          string
	password      string
	respKeyNT     []byte
	respKeyLM     []byte
	enableUnicode bool
	negotiateMsg  []byte
	challengeMsg  *ChallengeMessage
	authMsg       []byte
}

// NewNTLMv2 creates a new NTLMv2 authentication context
func NewNTLMv2(domain, user, password string) *NTLMv2 {
	n := &NTLMv2{
		domain:   domain,
		user:     user,
		password: password,
	}
	n.respKeyNT = ntowfv2(password, user, domain)
	n.respKeyLM = lmowfv2(password, user, domain)
	return n
}

// GetNegotiateMessage returns the NTLM Type 1 (Negotiate) message
func (n *NTLMv2) GetNegotiateMessage() []byte {
	flags := uint32(
		NTLMSSP_NEGOTIATE_KEY_EXCH |
			NTLMSSP_NEGOTIATE_128 |
			NTLMSSP_NEGOTIATE_EXTENDED_SESSIONSECURITY |
			NTLMSSP_NEGOTIATE_ALWAYS_SIGN |
			NTLMSSP_NEGOTIATE_NTLM |
			NTLMSSP_NEGOTIATE_SEAL |
			NTLMSSP_NEGOTIATE_SIGN |
			NTLMSSP_REQUEST_TARGET |
			NTLMSSP_NEGOTIATE_UNICODE |
			NTLMSSP_NEGOTIATE_VERSION)

	buf := &bytes.Buffer{}
	buf.Write(ntlmSignature)
	_ = binary.Write(buf, binary.LittleEndian, uint32(1)) // MessageType
	_ = binary.Write(buf, binary.LittleEndian, flags)
	// DomainNameFields (8 bytes - all zeros)
	_ = binary.Write(buf, binary.LittleEndian, uint16(0)) // DomainNameLen
	_ = binary.Write(buf, binary.LittleEndian, uint16(0)) // DomainNameMaxLen
	_ = binary.Write(buf, binary.LittleEndian, uint32(0)) // DomainNameBufferOffset
	// WorkstationFields (8 bytes - all zeros)
	_ = binary.Write(buf, binary.LittleEndian, uint16(0)) // WorkstationLen
	_ = binary.Write(buf, binary.LittleEndian, uint16(0)) // WorkstationMaxLen
	_ = binary.Write(buf, binary.LittleEndian, uint32(0)) // WorkstationBufferOffset
	// Version (8 bytes)
	buf.Write([]byte{0x06, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0F}) // Windows Vista, NTLMSSP_REVISION_W2K3

	n.negotiateMsg = buf.Bytes()
	return n.negotiateMsg
}

// ChallengeMessage represents NTLM Type 2 message
type ChallengeMessage struct {
	NegotiateFlags  uint32
	ServerChallenge [8]byte
	TargetInfo      []byte
	Timestamp       []byte
	RawData         []byte // Original raw data for MIC computation
}

// ParseChallengeMessage parses an NTLM Type 2 (Challenge) message
func ParseChallengeMessage(data []byte) (*ChallengeMessage, error) {
	if len(data) < 56 {
		return nil, bytes.ErrTooLarge
	}

	// Store raw data for MIC computation
	rawData := make([]byte, len(data))
	copy(rawData, data)

	// Skip signature (8) + messageType (4)
	offset := 12

	// TargetNameFields
	targetNameLen := binary.LittleEndian.Uint16(data[offset:])
	offset += 2
	offset += 2 // MaxLen
	targetNameOffset := binary.LittleEndian.Uint32(data[offset:])
	offset += 4
	_ = targetNameLen
	_ = targetNameOffset

	// NegotiateFlags
	flags := binary.LittleEndian.Uint32(data[offset:])
	offset += 4

	// ServerChallenge
	var challenge [8]byte
	copy(challenge[:], data[offset:offset+8])
	offset += 8

	// Reserved (8 bytes)
	offset += 8

	// TargetInfoFields
	targetInfoLen := binary.LittleEndian.Uint16(data[offset:])
	offset += 2
	offset += 2 // MaxLen
	targetInfoOffset := binary.LittleEndian.Uint32(data[offset:])

	msg := &ChallengeMessage{
		NegotiateFlags:  flags,
		ServerChallenge: challenge,
		RawData:         rawData,
	}

	// Extract target info if present
	if targetInfoLen > 0 && int(targetInfoOffset)+int(targetInfoLen) <= len(data) {
		msg.TargetInfo = data[targetInfoOffset : targetInfoOffset+uint32(targetInfoLen)]
		msg.Timestamp = extractTimestamp(msg.TargetInfo)
	}

	return msg, nil
}

func extractTimestamp(targetInfo []byte) []byte {
	offset := 0
	for offset+4 <= len(targetInfo) {
		avID := binary.LittleEndian.Uint16(targetInfo[offset:])
		avLen := binary.LittleEndian.Uint16(targetInfo[offset+2:])
		offset += 4

		if avID == MsvAvEOL {
			break
		}
		if avID == MsvAvTimestamp && avLen == 8 && offset+8 <= len(targetInfo) {
			return targetInfo[offset : offset+8]
		}
		offset += int(avLen)
	}
	return nil
}

// modifyTargetInfoForMIC modifies TargetInfo to add MsvAvFlags with MIC_PROVIDED flag
// Per MS-NLMP 3.1.5.1.2: When MIC is present, MsvAvFlags MUST have MIC_PROVIDED (0x02)
func modifyTargetInfoForMIC(targetInfo []byte) []byte {
	if len(targetInfo) == 0 {
		return targetInfo
	}

	// Find MsvAvFlags and MsvAvEOL positions
	flagsOffset := -1
	eolOffset := -1
	offset := 0

	for offset+4 <= len(targetInfo) {
		avID := binary.LittleEndian.Uint16(targetInfo[offset:])
		avLen := binary.LittleEndian.Uint16(targetInfo[offset+2:])

		if avID == MsvAvFlags {
			flagsOffset = offset
		}
		if avID == MsvAvEOL {
			eolOffset = offset
			break
		}
		offset += 4 + int(avLen)
	}

	result := make([]byte, len(targetInfo))
	copy(result, targetInfo)

	if flagsOffset >= 0 {
		// Update existing flags to include MIC_PROVIDED (0x02)
		existingFlags := binary.LittleEndian.Uint32(result[flagsOffset+4:])
		existingFlags |= 0x02 // MIC_PROVIDED
		binary.LittleEndian.PutUint32(result[flagsOffset+4:], existingFlags)
	} else if eolOffset >= 0 {
		// Insert MsvAvFlags before MsvAvEOL
		newPair := make([]byte, 8)
		binary.LittleEndian.PutUint16(newPair[0:], MsvAvFlags) // AvId
		binary.LittleEndian.PutUint16(newPair[2:], 4)          // AvLen
		binary.LittleEndian.PutUint32(newPair[4:], 0x02)       // MIC_PROVIDED

		result = append(result[:eolOffset], append(newPair, result[eolOffset:]...)...)
	}

	return result
}

// Security handles NTLM message encryption/decryption
type Security struct {
	encryptRC4 *rc4.Cipher
	decryptRC4 *rc4.Cipher
	signingKey []byte
	verifyKey  []byte
	seqNum     uint32
}

// GetAuthenticateMessage processes challenge and returns Type 3 message and security context
func (n *NTLMv2) GetAuthenticateMessage(challengeData []byte) ([]byte, *Security) {
	challenge, err := ParseChallengeMessage(challengeData)
	if err != nil {
		return nil, nil
	}
	n.challengeMsg = challenge

	if challenge.NegotiateFlags&NTLMSSP_NEGOTIATE_UNICODE != 0 {
		n.enableUnicode = true
	}

	// Get timestamp (use server's if available, else generate)
	var timestamp []byte
	computeMIC := false
	if challenge.Timestamp != nil {
		timestamp = challenge.Timestamp
		computeMIC = true
	} else {
		timestamp = makeTimestamp()
	}

	// Generate client challenge
	clientChallenge := make([]byte, 8)
	if _, err := rand.Read(clientChallenge); err != nil {
		return nil, nil
	}

	// Modify TargetInfo to include MIC_PROVIDED flag when MIC will be computed
	targetInfo := challenge.TargetInfo
	if computeMIC {
		targetInfo = modifyTargetInfoForMIC(challenge.TargetInfo)
	}

	// Compute responses
	ntChallengeResponse, lmChallengeResponse, sessionBaseKey := n.computeResponseV2(
		challenge.ServerChallenge[:], clientChallenge, timestamp, targetInfo)

	// Key exchange
	exportedSessionKey := make([]byte, 16)
	if _, err := rand.Read(exportedSessionKey); err != nil {
		return nil, nil
	}

	encryptedRandomSessionKey := make([]byte, 16)
	rc, _ := rc4.NewCipher(sessionBaseKey)
	rc.XORKeyStream(encryptedRandomSessionKey, exportedSessionKey)

	// Build authenticate message
	domain, user, _ := n.GetEncodedCredentials()
	workstation := []byte{}

	authMsg := n.buildAuthenticateMessage(
		challenge.NegotiateFlags,
		domain, user, workstation,
		lmChallengeResponse, ntChallengeResponse,
		encryptedRandomSessionKey, computeMIC)

	// Compute MIC if needed
	if computeMIC {
		mic := n.computeMIC(exportedSessionKey, authMsg)
		// MIC is at offset 72 in the authenticate message
		copy(authMsg[72:88], mic)
	}

	n.authMsg = authMsg

	// Create security context
	// Per MS-NLMP, with Extended Session Security, keys are derived using MD5
	// SignKey = MD5(SessionBaseKey || MagicConstant)
	// SealKey = MD5(SessionBaseKey || MagicConstant)
	clientSigningKey := md5Hash(append(exportedSessionKey, append([]byte("session key to client-to-server signing key magic constant"), 0x00)...))
	serverSigningKey := md5Hash(append(exportedSessionKey, append([]byte("session key to server-to-client signing key magic constant"), 0x00)...))
	clientSealingKey := md5Hash(append(exportedSessionKey, append([]byte("session key to client-to-server sealing key magic constant"), 0x00)...))
	serverSealingKey := md5Hash(append(exportedSessionKey, append([]byte("session key to server-to-client sealing key magic constant"), 0x00)...))

	encryptRC4, _ := rc4.NewCipher(clientSealingKey)
	decryptRC4, _ := rc4.NewCipher(serverSealingKey)

	return authMsg, &Security{
		encryptRC4: encryptRC4,
		decryptRC4: decryptRC4,
		signingKey: clientSigningKey,
		verifyKey:  serverSigningKey,
		seqNum:     0,
	}
}

func (n *NTLMv2) computeResponseV2(serverChallenge, clientChallenge, timestamp, targetInfo []byte) ([]byte, []byte, []byte) {
	// Build temp
	temp := &bytes.Buffer{}
	temp.Write([]byte{0x01, 0x01})                         // RespType, HiRespType
	temp.Write([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00}) // Reserved
	temp.Write(timestamp)                                  // TimeStamp
	temp.Write(clientChallenge)                            // ClientChallenge
	temp.Write([]byte{0x00, 0x00, 0x00, 0x00})             // Reserved
	temp.Write(targetInfo)                                 // AvPairs
	temp.Write([]byte{0x00, 0x00, 0x00, 0x00})             // Reserved

	// NTProofStr = HMAC_MD5(ResponseKeyNT, ServerChallenge || temp)
	ntBuf := append(serverChallenge, temp.Bytes()...)
	ntProofStr := hmacMD5(n.respKeyNT, ntBuf)

	// NtChallengeResponse = NTProofStr || temp
	ntChallengeResponse := append(ntProofStr, temp.Bytes()...)

	// LmChallengeResponse = HMAC_MD5(ResponseKeyLM, ServerChallenge || ClientChallenge) || ClientChallenge
	lmBuf := append(serverChallenge, clientChallenge...)
	lmChallengeResponse := append(hmacMD5(n.respKeyLM, lmBuf), clientChallenge...)

	// SessionBaseKey = HMAC_MD5(ResponseKeyNT, NTProofStr)
	sessionBaseKey := hmacMD5(n.respKeyNT, ntProofStr)

	return ntChallengeResponse, lmChallengeResponse, sessionBaseKey
}

func (n *NTLMv2) buildAuthenticateMessage(flags uint32, domain, user, workstation, lmResponse, ntResponse, encryptedKey []byte, includeMIC bool) []byte {
	// Calculate payload offset
	payloadOffset := uint32(88) // Fixed header size including MIC

	buf := &bytes.Buffer{}
	buf.Write(ntlmSignature)
	_ = binary.Write(buf, binary.LittleEndian, uint32(3)) // MessageType

	currentOffset := payloadOffset

	// LmChallengeResponseFields
	_ = binary.Write(buf, binary.LittleEndian, uint16(len(lmResponse)))
	_ = binary.Write(buf, binary.LittleEndian, uint16(len(lmResponse)))
	_ = binary.Write(buf, binary.LittleEndian, currentOffset)
	currentOffset += uint32(len(lmResponse))

	// NtChallengeResponseFields
	_ = binary.Write(buf, binary.LittleEndian, uint16(len(ntResponse)))
	_ = binary.Write(buf, binary.LittleEndian, uint16(len(ntResponse)))
	_ = binary.Write(buf, binary.LittleEndian, currentOffset)
	currentOffset += uint32(len(ntResponse))

	// DomainNameFields
	_ = binary.Write(buf, binary.LittleEndian, uint16(len(domain)))
	_ = binary.Write(buf, binary.LittleEndian, uint16(len(domain)))
	_ = binary.Write(buf, binary.LittleEndian, currentOffset)
	currentOffset += uint32(len(domain))

	// UserNameFields
	_ = binary.Write(buf, binary.LittleEndian, uint16(len(user)))
	_ = binary.Write(buf, binary.LittleEndian, uint16(len(user)))
	_ = binary.Write(buf, binary.LittleEndian, currentOffset)
	currentOffset += uint32(len(user))

	// WorkstationFields
	_ = binary.Write(buf, binary.LittleEndian, uint16(len(workstation)))
	_ = binary.Write(buf, binary.LittleEndian, uint16(len(workstation)))
	_ = binary.Write(buf, binary.LittleEndian, currentOffset)
	currentOffset += uint32(len(workstation))

	// EncryptedRandomSessionKeyFields
	_ = binary.Write(buf, binary.LittleEndian, uint16(len(encryptedKey)))
	_ = binary.Write(buf, binary.LittleEndian, uint16(len(encryptedKey)))
	_ = binary.Write(buf, binary.LittleEndian, currentOffset)

	// NegotiateFlags
	_ = binary.Write(buf, binary.LittleEndian, flags)

	// Version (8 bytes)
	buf.Write([]byte{0x06, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0F})

	// MIC (16 bytes - zeros for now, filled in later if needed)
	buf.Write(make([]byte, 16))

	// Payload
	buf.Write(lmResponse)
	buf.Write(ntResponse)
	buf.Write(domain)
	buf.Write(user)
	buf.Write(workstation)
	buf.Write(encryptedKey)

	return buf.Bytes()
}

func (n *NTLMv2) computeMIC(exportedSessionKey, authMsg []byte) []byte {
	buf := &bytes.Buffer{}
	buf.Write(n.negotiateMsg)
	// Use raw challenge data for MIC computation
	buf.Write(n.challengeMsg.RawData)
	// Write authMsg with zeroed MIC field
	micZeroed := make([]byte, len(authMsg))
	copy(micZeroed, authMsg)
	for i := 72; i < 88 && i < len(micZeroed); i++ {
		micZeroed[i] = 0
	}
	buf.Write(micZeroed)
	return hmacMD5(exportedSessionKey, buf.Bytes())[:16]
}

// GetEncodedCredentials returns domain, user, password encoded appropriately for NTLM
func (n *NTLMv2) GetEncodedCredentials() ([]byte, []byte, []byte) {
	if n.enableUnicode {
		return unicodeEncode(n.domain), unicodeEncode(n.user), unicodeEncode(n.password)
	}
	return []byte(n.domain), []byte(n.user), []byte(n.password)
}

// GetCredSSPCredentials returns domain, user, password as UTF-16LE for CredSSP TSCredentials
// Per MS-CSSP, TSPasswordCreds MUST always use UTF-16LE encoding
func (n *NTLMv2) GetCredSSPCredentials() ([]byte, []byte, []byte) {
	return unicodeEncode(n.domain), unicodeEncode(n.user), unicodeEncode(n.password)
}

// GssEncrypt encrypts data using NTLM seal
// Per MS-NLMP: First encrypt data, THEN compute and encrypt signature
func (s *Security) GssEncrypt(data []byte) []byte {
	// Step 1: Encrypt the data FIRST
	encrypted := make([]byte, len(data))
	s.encryptRC4.XORKeyStream(encrypted, data)

	// Step 2: Compute signature over ORIGINAL plaintext (not encrypted)
	seqBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(seqBuf, s.seqNum)

	signBuf := &bytes.Buffer{}
	signBuf.Write(seqBuf)
	signBuf.Write(data) // Use original plaintext for signature
	sig := hmacMD5(s.signingKey, signBuf.Bytes())[:8]

	// Step 3: Encrypt the signature checksum (RC4 state continues from data encryption)
	checksum := make([]byte, 8)
	s.encryptRC4.XORKeyStream(checksum, sig)

	// Build GSS token: Version(4) + Checksum(8) + SeqNum(4) + EncryptedData
	result := &bytes.Buffer{}
	_ = binary.Write(result, binary.LittleEndian, uint32(0x00000001)) // Version
	result.Write(checksum)
	_ = binary.Write(result, binary.LittleEndian, s.seqNum)
	result.Write(encrypted)

	s.seqNum++
	return result.Bytes()
}

// GssDecrypt decrypts data using NTLM unseal
// Input format: Version(4) + Checksum(8) + SeqNum(4) + EncryptedData
func (s *Security) GssDecrypt(data []byte) []byte {
	if len(data) < 16 {
		return nil
	}

	// Parse signature
	version := binary.LittleEndian.Uint32(data[0:4])
	if version != 1 {
		return nil // Invalid version
	}
	receivedChecksum := data[4:12]
	receivedSeqNum := binary.LittleEndian.Uint32(data[12:16])
	encrypted := data[16:]

	// Decrypt the data
	decrypted := make([]byte, len(encrypted))
	s.decryptRC4.XORKeyStream(decrypted, encrypted)

	// Verify checksum: compute expected checksum over decrypted plaintext
	seqBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(seqBuf, receivedSeqNum)

	signBuf := &bytes.Buffer{}
	signBuf.Write(seqBuf)
	signBuf.Write(decrypted)
	expectedSig := hmacMD5(s.verifyKey, signBuf.Bytes())[:8]

	// Encrypt the expected signature (RC4 state continues from data decryption)
	expectedChecksum := make([]byte, 8)
	s.decryptRC4.XORKeyStream(expectedChecksum, expectedSig)

	// Compare checksums (constant-time comparison to prevent timing attacks)
	if !hmac.Equal(receivedChecksum, expectedChecksum) {
		return nil // Checksum verification failed
	}

	return decrypted
}

// Helper functions

func unicodeEncode(s string) []byte {
	runes := utf16.Encode([]rune(s))
	result := make([]byte, len(runes)*2)
	for i, r := range runes {
		binary.LittleEndian.PutUint16(result[i*2:], r)
	}
	return result
}

func ntowfv2(password, user, domain string) []byte {
	// NTOWFv2 = HMAC_MD5(MD4(UNICODE(Password)), UNICODE(ConcatenationOf(Uppercase(User), Domain)))
	passHash := md4(unicodeEncode(password))
	concat := unicodeEncode(toUpper(user) + domain)
	return hmacMD5(passHash, concat)
}

func lmowfv2(password, user, domain string) []byte {
	// LMOWFv2 = NTOWFv2 (same computation)
	return ntowfv2(password, user, domain)
}

func hmacMD5(key, data []byte) []byte {
	h := hmac.New(md5.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func md5Hash(data []byte) []byte {
	h := md5.Sum(data)
	return h[:]
}

func makeTimestamp() []byte {
	// Windows FILETIME: 100-nanosecond intervals since January 1, 1601
	ft := uint64(time.Now().UnixNano())/100 + 116444736000000000
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, ft)
	return buf
}

func toUpper(s string) string {
	result := make([]rune, len(s))
	for i, r := range s {
		if r >= 'a' && r <= 'z' {
			result[i] = r - 32
		} else {
			result[i] = r
		}
	}
	return string(result)
}
