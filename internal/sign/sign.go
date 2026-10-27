package sign

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"

	"gateway-control/internal/model"
)

type Signer struct {
	private ed25519.PrivateKey
	keyID   string
}

func LoadPrivateKey(path string) (Signer, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Signer{}, err
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return Signer{}, fmt.Errorf("imza anahtarı PEM değil")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return Signer{}, err
	}
	key, ok := parsed.(ed25519.PrivateKey)
	if !ok {
		return Signer{}, fmt.Errorf("imza anahtarı Ed25519 değil")
	}
	return Signer{private: key, keyID: KeyID(key.Public().(ed25519.PublicKey))}, nil
}

func LoadPublicKey(path string) (ed25519.PublicKey, string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, "", fmt.Errorf("açık anahtar PEM değil")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, "", err
	}
	key, ok := parsed.(ed25519.PublicKey)
	if !ok {
		return nil, "", fmt.Errorf("açık anahtar Ed25519 değil")
	}
	return key, KeyID(key), nil
}

func KeyID(public ed25519.PublicKey) string {
	sum := sha256.Sum256(public)
	return hex.EncodeToString(sum[:8])
}

func (s Signer) KeyID() string { return s.keyID }

func (s Signer) Sign(payload []byte) (model.Envelope, error) {
	sum := sha256.Sum256(payload)
	signature := ed25519.Sign(s.private, payload)
	return model.Envelope{
		Payload:     append([]byte(nil), payload...),
		ContentHash: hex.EncodeToString(sum[:]),
		Signature:   base64.StdEncoding.EncodeToString(signature),
		KeyID:       s.keyID,
	}, nil
}

func Verify(public ed25519.PublicKey, env model.Envelope) error {
	sum := sha256.Sum256(env.Payload)
	if hex.EncodeToString(sum[:]) != env.ContentHash {
		return fmt.Errorf("içerik hash'i uyuşmuyor")
	}
	signature, err := base64.StdEncoding.DecodeString(env.Signature)
	if err != nil {
		return fmt.Errorf("imza çözülemedi")
	}
	if !ed25519.Verify(public, env.Payload, signature) {
		return fmt.Errorf("imza doğrulanamadı")
	}
	return nil
}

func GeneratePEM() (privatePEM, publicPEM []byte, err error) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	privBytes, err := x509.MarshalPKCS8PrivateKey(private)
	if err != nil {
		return nil, nil, err
	}
	pubBytes, err := x509.MarshalPKIXPublicKey(public)
	if err != nil {
		return nil, nil, err
	}
	privatePEM = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
	publicPEM = pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})
	return privatePEM, publicPEM, nil
}
