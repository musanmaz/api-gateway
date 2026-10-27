package store

import (
	"errors"
	"testing"

	"gateway-control/internal/validate"
)

func TestNormalizePublishNode(t *testing.T) {
	name, err := normalizePublishNode("  gw-local-1 ")
	if err != nil || name != "gw-local-1" {
		t.Fatalf("name %q err %v", name, err)
	}
	if _, err := normalizePublishNode(""); err != nil {
		t.Fatal(err)
	}
	_, err = normalizePublishNode("GW")
	var invalid *validate.Error
	if !errors.As(err, &invalid) {
		t.Fatalf("err %v", err)
	}
}
