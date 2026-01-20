# internal/auth

RDP authentication implementation supporting NTLMv2 and CredSSP/NLA.

## Overview

This package implements the complete authentication stack for RDP connections:
- **NTLMv2** - NT LAN Manager version 2 authentication protocol
- **CredSSP** - Credential Security Support Provider (MS-CSSP)
- **NLA** - Network Level Authentication (uses CredSSP internally)

It handles the full 3-message NTLM handshake plus message encryption/decryption for secure RDP sessions.

## Files

| File | Purpose |
|------|---------|
| `ntlm.go` | NTLMv2 protocol: negotiate/challenge/authenticate messages, signing, sealing |
| `credssp.go` | CredSSP protocol: TSRequest encoding/decoding, public key authentication |
| `md4.go` | MD4 hash implementation (required for NTLM password hashing) |
| `auth_test.go` | Unit tests for all authentication components |

## NTLMv2 Authentication Flow

```
┌─────────┐                                    ┌─────────┐
│ Client  │                                    │ Server  │
└────┬────┘                                    └────┬────┘
     │                                              │
     │  1. NEGOTIATE_MESSAGE (Type 1)               │
     │  ─────────────────────────────────────────►  │
     │  (Flags, Domain hint, Workstation)           │
     │                                              │
     │  2. CHALLENGE_MESSAGE (Type 2)               │
     │  ◄─────────────────────────────────────────  │
     │  (Server challenge, Target info, Timestamp)  │
     │                                              │
     │  3. AUTHENTICATE_MESSAGE (Type 3)            │
     │  ─────────────────────────────────────────►  │
     │  (NT response, LM response, Encrypted key)   │
     │                                              │
     │  4. Session established with encryption      │
     │  ◄────────────────────────────────────────►  │
```

## Key Structs

### NTLMv2

```go
type NTLMv2 struct {
    Domain, User, Password string    // Credentials
    NTResponse, LMResponse []byte    // Computed responses
    ExportedSessionKey     []byte    // Session encryption key
    NegotiateMessage       []byte    // Type 1 message
    ChallengeMessage       []byte    // Type 2 message (from server)
    AuthenticateMessage    []byte    // Type 3 message
}
```

### Security (Message Encryption)

```go
type Security struct {
    outgoingSigningKey, incomingSigningKey []byte
    outgoingSealingKey, incomingSealingKey []byte
    outgoingRC4, incomingRC4               *rc4.Cipher
    outgoingSeqNum, incomingSeqNum         uint32
}
```

### TSRequest (CredSSP)

```go
type TSRequest struct {
    Version       int      // Protocol version (2-6+)
    NegoTokens    []byte   // NTLM message wrapper
    AuthInfo      []byte   // Encrypted credentials
    PubKeyAuth    []byte   // Public key authentication
    ClientNonce   []byte   // Version 5+ nonce binding
    ErrorCode     int      // Error status
}
```

## Cryptographic Operations

### Password to NT Hash

```
NTOWFv2 = HMAC-MD5(
    MD4(UTF-16LE(Password)),
    UTF-16LE(UPPER(User) + Domain)
)
```

### Response Computation

```
NTProofStr = HMAC-MD5(
    ResponseKeyNT,
    ServerChallenge || ClientChallenge || Timestamp || TargetInfo
)
```

### Session Key Derivation

```
SessionBaseKey = HMAC-MD5(ResponseKeyNT, NTProofStr)
ExportedSessionKey = RC4(SessionBaseKey, EncryptedRandomSessionKey)
```

### Message Signing/Sealing

- **Signing**: HMAC-MD5 over sequence number + plaintext
- **Sealing**: RC4 stream cipher encryption
- **Sequence Numbers**: Prevents replay attacks

## CredSSP (NLA) Wrapper

CredSSP wraps NTLM messages with additional security:

1. **Version Negotiation** - Client/server agree on protocol version
2. **TLS Binding** - Version 5+ uses SHA256 hash of server certificate
3. **Nonce Exchange** - Prevents man-in-the-middle attacks

```go
// Encode TSRequest for wire transmission
func EncodeTSRequest(version int, negoTokens, authInfo, pubKeyAuth, clientNonce []byte) []byte

// Decode TSRequest from server
func DecodeTSRequest(data []byte) (*TSRequest, error)
```

## Usage

```go
// Create NTLM authenticator
ntlm := auth.NewNTLMv2("DOMAIN", "username", "password")

// Generate Type 1 (Negotiate) message
negotiate := ntlm.GetNegotiateMessage()

// Process Type 2 (Challenge) from server
security, err := ntlm.GetAuthenticateMessage(challengeData)

// Get Type 3 (Authenticate) message
authenticate := ntlm.AuthenticateMessage

// Encrypt/decrypt subsequent messages
encrypted := security.GssEncrypt(plaintext)
decrypted := security.GssDecrypt(ciphertext)
```

## Security Features

- **Extended Session Security** - MD5-derived signing/sealing keys
- **MIC (Message Integrity Code)** - Per MS-NLMP specification
- **Version 5+ Public Key Binding** - SHA256-based with nonce
- **Unicode Handling** - Proper UTF-16LE encoding
- **Replay Protection** - Sequence number tracking

## References

- **MS-NLMP** - NT LAN Manager (NTLM) Authentication Protocol
- **MS-CSSP** - Credential Security Support Provider Protocol
- **RFC 4178** - SPNEGO (Simple and Protected GSSAPI Negotiation)
