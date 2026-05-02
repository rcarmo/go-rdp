// Package auth exposes RDP authentication helpers from go-rdp.
package auth

import internalauth "github.com/rcarmo/go-rdp/internal/auth"

type TSRequest = internalauth.TSRequest
type NegoToken = internalauth.NegoToken
type NTLMv2 = internalauth.NTLMv2
type ChallengeMessage = internalauth.ChallengeMessage

var ClientServerHashMagic = internalauth.ClientServerHashMagic
var ServerClientHashMagic = internalauth.ServerClientHashMagic

const (
	NTLMSSP_NEGOTIATE_56                       = internalauth.NTLMSSP_NEGOTIATE_56
	NTLMSSP_NEGOTIATE_KEY_EXCH                 = internalauth.NTLMSSP_NEGOTIATE_KEY_EXCH
	NTLMSSP_NEGOTIATE_128                      = internalauth.NTLMSSP_NEGOTIATE_128
	NTLMSSP_NEGOTIATE_VERSION                  = internalauth.NTLMSSP_NEGOTIATE_VERSION
	NTLMSSP_NEGOTIATE_TARGET_INFO              = internalauth.NTLMSSP_NEGOTIATE_TARGET_INFO
	NTLMSSP_REQUEST_NON_NT_SESSION_KEY         = internalauth.NTLMSSP_REQUEST_NON_NT_SESSION_KEY
	NTLMSSP_NEGOTIATE_IDENTIFY                 = internalauth.NTLMSSP_NEGOTIATE_IDENTIFY
	NTLMSSP_NEGOTIATE_EXTENDED_SESSIONSECURITY = internalauth.NTLMSSP_NEGOTIATE_EXTENDED_SESSIONSECURITY
	NTLMSSP_TARGET_TYPE_SERVER                 = internalauth.NTLMSSP_TARGET_TYPE_SERVER
	NTLMSSP_TARGET_TYPE_DOMAIN                 = internalauth.NTLMSSP_TARGET_TYPE_DOMAIN
	NTLMSSP_NEGOTIATE_ALWAYS_SIGN              = internalauth.NTLMSSP_NEGOTIATE_ALWAYS_SIGN
	NTLMSSP_NEGOTIATE_NTLM                     = internalauth.NTLMSSP_NEGOTIATE_NTLM
	NTLMSSP_NEGOTIATE_SEAL                     = internalauth.NTLMSSP_NEGOTIATE_SEAL
	NTLMSSP_NEGOTIATE_SIGN                     = internalauth.NTLMSSP_NEGOTIATE_SIGN
	NTLMSSP_REQUEST_TARGET                     = internalauth.NTLMSSP_REQUEST_TARGET
	NTLMSSP_NEGOTIATE_UNICODE                  = internalauth.NTLMSSP_NEGOTIATE_UNICODE
)

func NewNTLMv2(domain, user, password string) *NTLMv2 {
	return internalauth.NewNTLMv2(domain, user, password)
}
func ParseChallengeMessage(data []byte) (*ChallengeMessage, error) {
	return internalauth.ParseChallengeMessage(data)
}
func ComputeClientPubKeyAuth(version int, pubKey, nonce []byte) []byte {
	return internalauth.ComputeClientPubKeyAuth(version, pubKey, nonce)
}
func VerifyServerPubKeyAuth(version int, serverPubKeyAuth, clientPubKey, nonce []byte) bool {
	return internalauth.VerifyServerPubKeyAuth(version, serverPubKeyAuth, clientPubKey, nonce)
}
func EncodeTSRequest(ntlmMessages [][]byte, authInfo []byte, pubKeyAuth []byte) []byte {
	return internalauth.EncodeTSRequest(ntlmMessages, authInfo, pubKeyAuth)
}
func EncodeTSRequestWithNonce(ntlmMessages [][]byte, authInfo []byte, pubKeyAuth []byte, clientNonce []byte) []byte {
	return internalauth.EncodeTSRequestWithNonce(ntlmMessages, authInfo, pubKeyAuth, clientNonce)
}
func EncodeTSRequestWithVersion(version int, ntlmMessages [][]byte, authInfo []byte, pubKeyAuth []byte, clientNonce []byte) []byte {
	return internalauth.EncodeTSRequestWithVersion(version, ntlmMessages, authInfo, pubKeyAuth, clientNonce)
}
func DecodeTSRequest(data []byte) (*TSRequest, error) { return internalauth.DecodeTSRequest(data) }
func EncodeCredentials(domain, username, password []byte) []byte {
	return internalauth.EncodeCredentials(domain, username, password)
}
