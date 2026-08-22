package sources

import "testing"

func TestNormalizeHost(t *testing.T) {
	cases := map[string]string{
		"127.0.0.1:8181":                 "127.0.0.1:8181",
		"ws://127.0.0.1:8181/ws/sso":     "127.0.0.1:8181",
		"wss://guild.example.com:8181/":  "guild.example.com:8181",
		" something.domain.com:8181 ":    "something.domain.com:8181",
	}
	for in, want := range cases {
		if got := NormalizeHost(in); got != want {
			t.Fatalf("NormalizeHost(%q)=%q want %q", in, got, want)
		}
	}
}

func TestWebSocketURL(t *testing.T) {
	got, err := WebSocketURL("127.0.0.1:8181")
	if err != nil || got != "ws://127.0.0.1:8181/ws/sso" {
		t.Fatalf("loopback: got %q err %v", got, err)
	}
	got, err = WebSocketURL("guild.example.com:8181")
	if err != nil || got != "wss://guild.example.com:8181/ws/sso" {
		t.Fatalf("public: got %q err %v", got, err)
	}
	got, err = WebSocketURL("ws://guild.example.com:8181")
	if err != nil || got != "ws://guild.example.com:8181/ws/sso" {
		t.Fatalf("override: got %q err %v", got, err)
	}
	got, err = WebSocketURL("http://guild.example.com:8181")
	if err != nil || got != "ws://guild.example.com:8181/ws/sso" {
		t.Fatalf("http: got %q err %v", got, err)
	}
	got, err = WebSocketURL("https://guild.example.com:8181")
	if err != nil || got != "wss://guild.example.com:8181/ws/sso" {
		t.Fatalf("https: got %q err %v", got, err)
	}
	got, err = WebSocketURL("10.0.0.5:8181")
	if err != nil || got != "ws://10.0.0.5:8181/ws/sso" {
		t.Fatalf("private: got %q err %v", got, err)
	}
	got, err = WebSocketURL("localhost:8181")
	if err != nil || got != "ws://localhost:8181/ws/sso" {
		t.Fatalf("localhost: got %q err %v", got, err)
	}
	got, err = WebSocketURL("guild.local:8181")
	if err != nil || got != "ws://guild.local:8181/ws/sso" {
		t.Fatalf(".local: got %q err %v", got, err)
	}
	got, err = WebSocketURL("wss://127.0.0.1:8181")
	if err != nil || got != "wss://127.0.0.1:8181/ws/sso" {
		t.Fatalf("explicit wss: got %q err %v", got, err)
	}
	if _, err := WebSocketURL(""); err == nil {
		t.Fatal("empty")
	}
}

func TestHostFromLegacyURLAndDialURL(t *testing.T) {
	if got := HostFromLegacyURL("ws://10.0.0.5:8181/ws/sso"); got != "10.0.0.5:8181" {
		t.Fatalf("%q", got)
	}
	if got := HostFromLegacyURL(""); got != "" {
		t.Fatalf("%q", got)
	}
	if got := HostFromLegacyURL("not a url ://"); got == "" {
		// NormalizeHost may still return something; just exercise fallback path
		_ = got
	}
	if got := HostFromLegacyURL("plain-host:8181"); got != "plain-host:8181" {
		t.Fatalf("fallback %q", got)
	}
	src := Source{Host: "guild.example.com:8181"}
	got, err := src.DialURL()
	if err != nil || got != "wss://guild.example.com:8181/ws/sso" {
		t.Fatalf("%q %v", got, err)
	}
	legacy := Source{URL: "ws://127.0.0.1:8181/ws/sso"}
	got, err = legacy.DialURL()
	if err != nil || got != "ws://127.0.0.1:8181/ws/sso" {
		t.Fatalf("legacy dial %q %v", got, err)
	}
}

func TestParseImportSources(t *testing.T) {
	one, err := ParseImportSources([]byte(`{"name":"Guild","host":"guild.example.com:8181","notes":"hi"}`))
	if err != nil || len(one) != 1 || one[0].Name != "Guild" || one[0].Host != "guild.example.com:8181" {
		t.Fatalf("one: %+v err=%v", one, err)
	}
	wrapped, err := ParseImportSources([]byte(`{"source":{"name":"A","host":"127.0.0.1:8181","token":"tok"}}`))
	if err != nil || len(wrapped) != 1 || wrapped[0].Token != "tok" {
		t.Fatalf("wrapped: %+v err=%v", wrapped, err)
	}
	many, err := ParseImportSources([]byte(`{"sources":[{"name":"A","host":"a.example:1"},{"name":"B","host":"b.example:2"}]}`))
	if err != nil || len(many) != 2 {
		t.Fatalf("many: %+v err=%v", many, err)
	}
	arr, err := ParseImportSources([]byte(`[{"name":"A","host":"127.0.0.1:8181"}]`))
	if err != nil || len(arr) != 1 {
		t.Fatalf("arr: %+v err=%v", arr, err)
	}
	if _, err := ParseImportSources([]byte(`{"host":"x"}`)); err == nil {
		t.Fatal("expected name required")
	}
}

func TestNormalizeConnectionMode(t *testing.T) {
	cases := map[string]ConnectionMode{
		"":           ConnectionDisabled,
		"bogus":      ConnectionDisabled,
		"login_sso":  ConnectionLoginSSO,
		"login_only": ConnectionLoginOnly,
		"disabled":   ConnectionDisabled,
	}
	for in, want := range cases {
		if got := NormalizeConnectionMode(ConnectionMode(in)); got != want {
			t.Fatalf("%q → %q want %q", in, got, want)
		}
	}
	if !ConnectionLoginSSO.WantsSSO() || ConnectionLoginOnly.WantsSSO() {
		t.Fatal("WantsSSO")
	}
	if !ConnectionLoginSSO.WantsProxy() || !ConnectionLoginOnly.WantsProxy() || ConnectionDisabled.WantsProxy() {
		t.Fatal("WantsProxy")
	}
}
