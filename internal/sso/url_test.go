package sso

import "testing"

func TestNormalizeDialURL(t *testing.T) {
	u, err := normalizeDialURL("ws://evil.example:8181/ws/sso")
	if err == nil || u != nil {
		t.Fatal("expected remote ws rejected")
	}
	u, err = normalizeDialURL("http://127.0.0.1:8181/ws/sso")
	if err != nil || u.Scheme != "ws" {
		t.Fatalf("%v %#v", err, u)
	}
	u, err = normalizeDialURL("https://guild.example.com/ws/sso")
	if err != nil || u.Scheme != "wss" {
		t.Fatalf("%v %#v", err, u)
	}
	u, err = normalizeDialURL("ws://localhost:8181/ws/sso")
	if err != nil || u.Hostname() != "localhost" {
		t.Fatalf("%v %#v", err, u)
	}
	u, err = normalizeDialURL("wss://guild.example.com/ws/sso")
	if err != nil || u.Scheme != "wss" {
		t.Fatalf("%v %#v", err, u)
	}
	if _, err := normalizeDialURL("://bad"); err == nil {
		t.Fatal("expected parse error")
	}
}
