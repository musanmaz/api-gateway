package sign

import (
	"os"
	"path/filepath"
	"testing"

	"gateway-control/internal/model"
)

func TestSignAndRejectTamper(t *testing.T) {
	dir := t.TempDir()
	priv, pub, err := GeneratePEM()
	if err != nil {
		t.Fatal(err)
	}
	privPath := filepath.Join(dir, "key.pem")
	pubPath := filepath.Join(dir, "key.pub")
	if err := os.WriteFile(privPath, priv, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pubPath, pub, 0o644); err != nil {
		t.Fatal(err)
	}
	signer, err := LoadPrivateKey(privPath)
	if err != nil {
		t.Fatal(err)
	}
	public, _, err := LoadPublicKey(pubPath)
	if err != nil {
		t.Fatal(err)
	}
	env, err := signer.Sign([]byte(`{"revision":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := Verify(public, env); err != nil {
		t.Fatal(err)
	}
	env.Payload = []byte(`{"revision":2}`)
	if err := Verify(public, env); err == nil {
		t.Fatal("tampered payload was accepted")
	}
	env, err = signer.Sign([]byte(`{"revision":1}`))
	if err != nil {
		t.Fatal(err)
	}
	env.Signature = "aaaa"
	if err := Verify(public, env); err == nil {
		t.Fatal("bad signature was accepted")
	}
	old := env
	env, err = signer.Sign([]byte(`{"revision":1}`))
	if err != nil {
		t.Fatal(err)
	}
	env.ContentHash = old.ContentHash
	env.Signature = "not-base64!!!"
	if err := Verify(public, model.Envelope{Payload: env.Payload, ContentHash: env.ContentHash, Signature: env.Signature}); err == nil {
		t.Fatal("invalid signature encoding was accepted")
	}
}
