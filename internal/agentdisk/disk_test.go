package agentdisk

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"gateway-control/internal/sign"
)

func TestLoadFallsBackAndRejectsEmpty(t *testing.T) {
	dir := t.TempDir()
	privPEM, pubPEM, err := sign.GeneratePEM()
	if err != nil {
		t.Fatal(err)
	}
	privPath := filepath.Join(dir, "key.pem")
	pubPath := filepath.Join(dir, "key.pub")
	if err := os.WriteFile(privPath, privPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pubPath, pubPEM, 0o644); err != nil {
		t.Fatal(err)
	}
	signer, err := sign.LoadPrivateKey(privPath)
	if err != nil {
		t.Fatal(err)
	}
	public, _, err := sign.LoadPublicKey(pubPath)
	if err != nil {
		t.Fatal(err)
	}
	disk, err := Open(dir, public)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := disk.Load(); err == nil {
		t.Fatal("empty disk was trusted")
	}
	first, err := signer.Sign(payload(1))
	if err != nil {
		t.Fatal(err)
	}
	first.Revision = 1
	if err := disk.Save(first); err != nil {
		t.Fatal(err)
	}
	second, err := signer.Sign(payload(2))
	if err != nil {
		t.Fatal(err)
	}
	second.Revision = 2
	if err := disk.Save(second); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "accepted", "2.json"), []byte("{bad"), 0o644); err != nil {
		t.Fatal(err)
	}
	env, pkg, err := disk.Load()
	if err != nil {
		t.Fatal(err)
	}
	if env.Revision != 1 || pkg.Revision != 1 {
		t.Fatalf("fallback revision = %d", env.Revision)
	}
	if err := os.WriteFile(filepath.Join(dir, "accepted", "1.json"), []byte("{bad"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := disk.Load(); err == nil {
		t.Fatal("corrupt cache was trusted")
	}
}

func payload(revision int) []byte {
	return []byte(`{"revision":` + strconv.Itoa(revision) + `,"fleetId":"fleet","issuedAt":"2026-10-22T00:00:00Z","listeners":[],"virtualHosts":[],"clusters":[]}`)
}
