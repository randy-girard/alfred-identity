package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alfred-identity/app/internal/localdata"
	"github.com/alfred-identity/app/internal/logbuf"
	"github.com/alfred-identity/app/internal/sources"
)

func testAppWithConfig(t *testing.T) (*App, string) {
	t.Helper()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	mgr, err := sources.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	local := &localdata.Store{
		AccountsPath:   filepath.Join(dir, "accounts.csv"),
		CharactersPath: filepath.Join(dir, "characters.csv"),
	}
	buf := logbuf.New(3000)
	a := &App{
		cfg:    mgr,
		local:  local,
		logBuf: buf,
		log:    slog.New(logbuf.NewHandler(buf, io.Discard)),
		ctx:    context.Background(),
	}
	return a, dir
}

func reserveUDPPort(t *testing.T) int {
	t.Helper()
	ln, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	port := ln.LocalAddr().(*net.UDPAddr).Port
	ln.Close()
	return port
}

func logsContain(a *App, needle string) bool {
	for _, e := range a.GetLogs(500) {
		if strings.Contains(e.Message, needle) {
			return true
		}
	}
	return false
}

func TestGetLogsAndClearLogs(t *testing.T) {
	a, _ := testAppWithConfig(t)
	a.log.Info("coverage probe")
	if len(a.GetLogs(0)) == 0 {
		t.Fatal("expected default limit logs")
	}
	a.ClearLogs()
	if len(a.GetLogs(100)) != 0 {
		t.Fatal("clear should empty buffer")
	}
}

func TestStartProxySkipsEqhostLogWhenUnchanged(t *testing.T) {
	a, dir := testAppWithConfig(t)
	upstream, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer upstream.Close()

	listenPort := reserveUDPPort(t)
	listenAddr := fmt.Sprintf("127.0.0.1:%d", listenPort)
	eqDir := filepath.Join(dir, "EverQuest")
	if err := os.MkdirAll(eqDir, 0o755); err != nil {
		t.Fatal(err)
	}
	eqhostContent := fmt.Sprintf("[LoginServer]\nHost=%s\n", listenAddr)
	if err := os.WriteFile(filepath.Join(eqDir, "eqhost.txt"), []byte(eqhostContent), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := a.cfg.Update(func(c *sources.Config) {
		c.EQDirectory = eqDir
		c.ListenAddr = listenAddr
		c.UpstreamAddr = upstream.LocalAddr().String()
	}); err != nil {
		t.Fatal(err)
	}

	if err := a.startProxy(false); err != nil {
		t.Fatal(err)
	}
	defer a.stopProxyRuntime(false)

	if logsContain(a, "eqhost updated") {
		t.Fatal("should not log eqhost update when host already matches")
	}
}

func TestStartProxyLogsEqhostWhenChanged(t *testing.T) {
	a, dir := testAppWithConfig(t)
	upstream, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer upstream.Close()

	listenPort := reserveUDPPort(t)
	listenAddr := fmt.Sprintf("127.0.0.1:%d", listenPort)
	eqDir := filepath.Join(dir, "EverQuest")
	if err := os.MkdirAll(eqDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(eqDir, "eqhost.txt"), []byte("[LoginServer]\nHost=login.example:5998\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := a.cfg.Update(func(c *sources.Config) {
		c.EQDirectory = eqDir
		c.ListenAddr = listenAddr
		c.UpstreamAddr = upstream.LocalAddr().String()
	}); err != nil {
		t.Fatal(err)
	}

	if err := a.startProxy(false); err != nil {
		t.Fatal(err)
	}
	defer a.stopProxyRuntime(false)

	if !logsContain(a, "eqhost updated") {
		t.Fatal("expected eqhost updated log when host changes")
	}
	cur, err := os.ReadFile(filepath.Join(eqDir, "eqhost.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cur), listenAddr) {
		t.Fatalf("eqhost not rewritten: %q", cur)
	}
}

func configureProxy(t *testing.T, a *App, dir string) (listenAddr string, cleanup func()) {
	t.Helper()
	upstream, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	listenPort := reserveUDPPort(t)
	listenAddr = fmt.Sprintf("127.0.0.1:%d", listenPort)
	if err := a.cfg.Update(func(c *sources.Config) {
		c.ListenAddr = listenAddr
		c.UpstreamAddr = upstream.LocalAddr().String()
		c.EQDirectory = ""
	}); err != nil {
		upstream.Close()
		t.Fatal(err)
	}
	return listenAddr, func() { upstream.Close() }
}

func TestSetListenAddrValidation(t *testing.T) {
	a, _ := testAppWithConfig(t)
	for _, addr := range []string{"", "noport", "10.0.0.1:6998"} {
		if err := a.SetListenAddr(addr); err == nil {
			t.Fatalf("expected error for %q", addr)
		}
	}
	port := reserveUDPPort(t)
	if err := a.SetListenAddr(fmt.Sprintf("127.0.0.1:%d", port)); err != nil {
		t.Fatal(err)
	}
	if a.cfg.Get().ListenAddr != fmt.Sprintf("127.0.0.1:%d", port) {
		t.Fatalf("got %q", a.cfg.Get().ListenAddr)
	}
}

func TestSetConnectionModeStartsAndStopsProxy(t *testing.T) {
	a, dir := testAppWithConfig(t)
	_, cleanup := configureProxy(t, a, dir)
	defer cleanup()

	if err := a.SetConnectionMode(string(sources.ConnectionLoginOnly)); err != nil {
		t.Fatal(err)
	}
	if a.proxy == nil {
		t.Fatal("expected proxy running")
	}
	if err := a.SetConnectionMode(string(sources.ConnectionDisabled)); err != nil {
		t.Fatal(err)
	}
	if a.proxy != nil {
		t.Fatal("expected proxy stopped")
	}
}
