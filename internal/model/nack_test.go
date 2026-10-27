package model

import "testing"

func TestNackDetailHidesKeyMaterial(t *testing.T) {
	if got := NackDetail("-----BEGIN PRIVATE KEY-----\nabc"); got != "ayrıntı gizlendi" {
		t.Fatalf("got %q", got)
	}
}

func TestNackDetailTruncates(t *testing.T) {
	got := NackDetail(stringsRepeat("a", 200))
	if len([]rune(got)) != 180 {
		t.Fatalf("len %d", len([]rune(got)))
	}
}

func stringsRepeat(value string, count int) string {
	out := ""
	for i := 0; i < count; i++ {
		out += value
	}
	return out
}
