package proxy

import (
	"context"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/alfred-identity/app/internal/router"
)

// Server is a UDP middleman between EQ client and the login server.
type Server struct {
	Listen   string
	Upstream string
	Router   *router.Router
	Log      *slog.Logger

	conn   *net.UDPConn
	mu     sync.Mutex
	cancel context.CancelFunc
}

func (s *Server) Start(parent context.Context) error {
	upAddr, err := net.ResolveUDPAddr("udp", s.Upstream)
	if err != nil {
		return err
	}

	log := s.Log
	if log == nil {
		log = slog.Default()
	}

	bindAddr := s.Listen
	if effective, changed := EffectiveBindAddr(s.Listen, upAddr); changed {
		log.Warn("loopback bind cannot reach external upstream; using effective bind address",
			"configured", s.Listen, "effective", effective, "upstream", s.Upstream)
		bindAddr = effective
	}

	addr, err := net.ResolveUDPAddr("udp", bindAddr)
	if err != nil {
		return err
	}
	c, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err // port in use → fail
	}

	ctx, cancel := context.WithCancel(parent)
	s.mu.Lock()
	s.conn = c
	s.cancel = cancel
	s.mu.Unlock()

	log.Info("UDP login proxy listening",
		"configured", s.Listen,
		"bound", c.LocalAddr().String(),
		"upstream", s.Upstream)

	engine := &Engine{Router: s.Router, Log: log}

	go func() {
		defer c.Close()
		buf := make([]byte, 65535)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			_ = c.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
			n, peer, err := c.ReadFromUDP(buf)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				continue
			}
			pkt := append([]byte{}, buf[:n]...)

			actions := engine.OnDatagram(ctx, pkt, peer, upAddr)
			client := engine.ClientAddr()

			for _, out := range engine.Finalize(actions.SendUpstream) {
				if _, err := c.WriteToUDP(out, upAddr); err != nil {
					log.Warn("send to upstream failed", "err", err)
				}
			}
			if client == nil {
				continue
			}
			for _, out := range engine.Finalize(actions.SendClient) {
				if _, err := c.WriteToUDP(out, client); err != nil {
					log.Warn("send to client failed", "err", err)
				}
			}
		}
	}()

	go func() {
		t := time.NewTicker(idleKeepaliveInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if pkt := engine.UpstreamKeepalivePacket(); pkt != nil {
					if _, err := c.WriteToUDP(pkt, upAddr); err != nil {
						log.Warn("idle keepalive to upstream failed", "err", err)
					}
				}
			}
		}
	}()
	return nil
}

func (s *Server) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	if s.conn != nil {
		_ = s.conn.Close()
		s.conn = nil
	}
}

func isUpstreamPeer(peer, upstream *net.UDPAddr) bool {
	if peer == nil || upstream == nil {
		return false
	}
	return normalizeAddr(peer).IP.Equal(normalizeAddr(upstream).IP) &&
		normalizeAddr(peer).Port == normalizeAddr(upstream).Port
}

func normalizeAddr(addr *net.UDPAddr) *net.UDPAddr {
	if addr == nil {
		return nil
	}
	if ip4 := addr.IP.To4(); ip4 != nil {
		return &net.UDPAddr{IP: ip4, Port: addr.Port, Zone: addr.Zone}
	}
	return addr
}
