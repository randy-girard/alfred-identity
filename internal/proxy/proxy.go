package proxy

import (
	"context"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/alfred-identity/app/internal/router"
)

// LoginRelay splices vault credentials on the daemon. EQ login UDP still
// originates from this machine so world transfer sees the player's IP.
type LoginRelay interface {
	Active() bool
	SpliceLogin(pkt []byte) ([]byte, error)
}

// Server is a UDP middleman between EQ client and the login server.
type Server struct {
	Listen   string
	Upstream string
	Router   *router.Router
	Log      *slog.Logger
	Relay    LoginRelay

	mu     sync.Mutex
	conn   *net.UDPConn
	engine *Engine
	upAddr *net.UDPAddr
	runCtx context.Context
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
	engine := &Engine{Router: s.Router, Log: log}
	s.mu.Lock()
	s.conn = c
	s.cancel = cancel
	s.engine = engine
	s.upAddr = upAddr
	s.runCtx = ctx
	s.mu.Unlock()

	via := "direct UDP to EQ login"
	if s.relayActive() {
		via = "direct UDP to EQ login (SSO vault splice)"
	}
	log.Info("UDP login proxy listening",
		"configured", s.Listen,
		"bound", c.LocalAddr().String(),
		"upstream", s.Upstream,
		"via", via)

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
			s.dispatch(engine, upAddr, actions)

			client := engine.ClientAddr()
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
					if err := s.sendUpstream(pkt); err != nil {
						log.Warn("idle keepalive to upstream failed", "err", err)
					}
				}
			}
		}
	}()
	return nil
}

func (s *Server) relayActive() bool {
	if s.Relay == nil {
		return false
	}
	return s.Relay.Active()
}

func (s *Server) dispatch(engine *Engine, upAddr *net.UDPAddr, actions Actions) {
	pkts := actions.SendUpstream
	if actions.SpliceSSO {
		if !s.relayActive() {
			if s.Log != nil {
				s.Log.Warn("SSO splice required but SSO is disconnected; not forwarding")
			}
			return
		}
		if len(pkts) == 0 {
			return
		}
		spliced, err := s.Relay.SpliceLogin(pkts[0])
		if err != nil {
			if s.Log != nil {
				s.Log.Warn("SSO splice failed", "err", err)
			}
			return
		}
		pkts = [][]byte{spliced}
	}
	for _, out := range engine.Finalize(pkts) {
		if err := s.sendUpstream(out); err != nil && s.Log != nil {
			s.Log.Warn("send to upstream failed", "err", err)
		}
	}
	_ = upAddr
}

func (s *Server) sendUpstream(pkt []byte) error {
	s.mu.Lock()
	c := s.conn
	up := s.upAddr
	s.mu.Unlock()
	if c == nil || up == nil {
		return net.ErrClosed
	}
	_, err := c.WriteToUDP(pkt, up)
	return err
}

// InjectFromLoginServer treats pkt as a datagram from the EQ login server
// (used when the daemon tunnels login_relay_down).
func (s *Server) InjectFromLoginServer(pkt []byte) {
	s.mu.Lock()
	c := s.conn
	engine := s.engine
	ctx := s.runCtx
	up := s.upAddr
	s.mu.Unlock()
	if c == nil || engine == nil || up == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	actions := engine.OnDatagram(ctx, pkt, up, up)
	client := engine.ClientAddr()
	if client == nil {
		return
	}
	for _, out := range engine.Finalize(actions.SendClient) {
		if _, err := c.WriteToUDP(out, client); err != nil && s.Log != nil {
			s.Log.Warn("send to client failed", "err", err)
		}
	}
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
	s.engine = nil
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
