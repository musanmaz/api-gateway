package httpapi

import "testing"

func TestCleanTryURL(t *testing.T) {
	cases := map[string]string{
		"http://127.0.0.1:10000":        "http://127.0.0.1:10000",
		" https://gw.example.com/ ":     "https://gw.example.com",
		"":                              "",
		"ftp://gw.example.com":          "",
		"http://user:pass@gw.example":   "",
		"http://gw.example/path":        "",
		"http://gw.example/?debug=true": "",
		"127.0.0.1:10000":               "",
	}
	for in, want := range cases {
		if got := cleanTryURL(in); got != want {
			t.Errorf("cleanTryURL(%q) = %q, want %q", in, got, want)
		}
	}
}
