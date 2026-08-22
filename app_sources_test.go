package main

import "testing"

func TestPreviewSourceJSON(t *testing.T) {
	a := &App{}
	got, err := a.PreviewSourceJSON(`{"name":"Guild","host":"127.0.0.1:8181","token":"tok"}`)
	if err != nil || len(got) != 1 {
		t.Fatalf("got=%#v err=%v", got, err)
	}
	if got[0].Name != "Guild" || got[0].Host != "127.0.0.1:8181" || got[0].Token != "tok" {
		t.Fatalf("%+v", got[0])
	}
	if _, err := a.PreviewSourceJSON(""); err == nil {
		t.Fatal("expected empty error")
	}
	if _, err := a.PreviewSourceJSON(`{"host":"x"}`); err == nil {
		t.Fatal("expected name required")
	}
}
