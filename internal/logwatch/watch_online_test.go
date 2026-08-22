package logwatch

import "testing"

func TestSetOnlineForTest(t *testing.T) {
	w := New(t.TempDir())
	w.SetOnlineForTest("Hero")
	got := w.OnlineCharacters()
	if len(got) != 1 || got[0] != "Hero" {
		t.Fatalf("%#v", got)
	}
	var nilWatcher *Watcher
	nilWatcher.SetOnlineForTest("x") // no panic
}
