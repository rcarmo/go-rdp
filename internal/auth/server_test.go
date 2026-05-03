package auth

import "testing"

func TestServerNTLMv2ChallengeAndVerify(t *testing.T) {
	client := NewNTLMv2("DOMAIN", "User", "Password")
	nego := client.GetNegotiateMessage()

	server, err := NewServerNTLMv2("DOMAIN", "SERVER")
	if err != nil {
		t.Fatal(err)
	}
	challenge, err := server.BuildChallengeMessage(nego)
	if err != nil {
		t.Fatal(err)
	}
	if parsed, err := ParseChallengeMessage(challenge); err != nil || parsed.NegotiateFlags&NTLMSSP_NEGOTIATE_TARGET_INFO == 0 {
		t.Fatalf("bad challenge flags parsed=%#v err=%v", parsed, err)
	}

	authMsg, clientSec := client.GetAuthenticateMessage(challenge)
	if authMsg == nil || clientSec == nil {
		t.Fatal("client failed to produce authenticate message")
	}
	parsedAuth, serverSec, err := server.VerifyAuthenticateMessage(authMsg, "User", "Password", "DOMAIN")
	if err != nil {
		t.Fatal(err)
	}
	if parsedAuth.User != "User" || parsedAuth.Domain != "DOMAIN" {
		t.Fatalf("unexpected parsed auth: %#v", parsedAuth)
	}

	sealed := clientSec.GssEncrypt([]byte("hello"))
	if got := string(serverSec.GssDecrypt(sealed)); got != "hello" {
		t.Fatalf("server decrypt = %q", got)
	}
	reply := serverSec.GssEncrypt([]byte("world"))
	if got := string(clientSec.GssDecrypt(reply)); got != "world" {
		t.Fatalf("client decrypt = %q", got)
	}
}

func TestServerNTLMv2RejectsBadPassword(t *testing.T) {
	client := NewNTLMv2("DOMAIN", "User", "Password")
	server, err := NewServerNTLMv2("DOMAIN", "SERVER")
	if err != nil {
		t.Fatal(err)
	}
	challenge, err := server.BuildChallengeMessage(client.GetNegotiateMessage())
	if err != nil {
		t.Fatal(err)
	}
	authMsg, _ := client.GetAuthenticateMessage(challenge)
	if _, _, err := server.VerifyAuthenticateMessage(authMsg, "User", "wrong", "DOMAIN"); err == nil {
		t.Fatal("expected bad password rejection")
	}
}

func TestDecodeCredentials(t *testing.T) {
	encoded := EncodeCredentials(unicodeEncode("DOMAIN"), unicodeEncode("User"), unicodeEncode("Password"))
	creds, err := DecodeCredentials(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if creds.Domain != "DOMAIN" || creds.Username != "User" || creds.Password != "Password" {
		t.Fatalf("unexpected credentials: %#v", creds)
	}
}

func TestComputeServerPubKeyAuth(t *testing.T) {
	pub := []byte{1, 2, 3}
	if got := ComputeServerPubKeyAuth(2, pub, nil); got[0] != 2 || got[1] != 2 || got[2] != 3 {
		t.Fatalf("unexpected v2 pubkey response: %x", got)
	}
	if got := ComputeServerPubKeyAuth(6, pub, []byte("nonce")); len(got) != 32 {
		t.Fatalf("unexpected v6 pubkey response length %d", len(got))
	}
}
